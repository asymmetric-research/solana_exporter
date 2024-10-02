package main

import (
	"context"
	"fmt"
	"time"

	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

const (
	slotPacerSchedule = 1 * time.Second
)

type SlotWatcher struct {
	client rpc.Provider

	leaderSlotAddresses []string

	// currentEpoch is the current epoch we are watching
	currentEpoch int64
	// firstSlot is the first slot [inclusive] of the current epoch which we are watching
	firstSlot int64
	// lastSlot is the last slot [inclusive] of the current epoch which we are watching
	lastSlot int64
	// slotWatermark is the last (most recent) slot we have tracked
	slotWatermark int64
}

var (
	totalTransactionsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "solana_confirmed_transactions_total",
		Help: "Total number of transactions processed since genesis (max confirmation)",
	})

	confirmedSlotHeight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "solana_confirmed_slot_height",
		Help: "Last confirmed slot height processed by watcher routine (max confirmation)",
	})

	currentEpochNumber = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "solana_confirmed_epoch_number",
		Help: "Current epoch (max confirmation)",
	})

	epochFirstSlot = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "solana_confirmed_epoch_first_slot",
		Help: "Current epoch's first slot (max confirmation)",
	})

	epochLastSlot = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "solana_confirmed_epoch_last_slot",
		Help: "Current epoch's last slot (max confirmation)",
	})

	leaderSlotsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "solana_leader_slots_total",
			Help: "(DEPRECATED) Number of leader slots per leader, grouped by skip status",
		},
		[]string{"status", "nodekey"},
	)

	leaderSlotsByEpoch = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "solana_leader_slots_by_epoch",
			Help: "Number of leader slots per leader, grouped by skip status and epoch",
		},
		[]string{"status", "nodekey", "epoch"},
	)
)

func NewCollectorSlotWatcher(collector *solanaCollector) *SlotWatcher {
	return &SlotWatcher{client: collector.rpcClient, leaderSlotAddresses: collector.leaderSlotAddresses}
}

func init() {
	prometheus.MustRegister(totalTransactionsTotal)
	prometheus.MustRegister(confirmedSlotHeight)
	prometheus.MustRegister(currentEpochNumber)
	prometheus.MustRegister(epochFirstSlot)
	prometheus.MustRegister(epochLastSlot)
	prometheus.MustRegister(leaderSlotsTotal)
	prometheus.MustRegister(leaderSlotsByEpoch)
}

func (c *SlotWatcher) WatchSlots(ctx context.Context, pace time.Duration) {
	ticker := time.NewTicker(pace)
	defer ticker.Stop()

	klog.Infof("Starting slot watcher")

	for {
		select {
		case <-ctx.Done():
			klog.Infof("Stopping WatchSlots() at slot %v", c.slotWatermark)
			return
		default:
			<-ticker.C

			ctx_, cancel := context.WithTimeout(ctx, httpTimeout)
			epochInfo, err := c.client.GetEpochInfo(ctx_, rpc.CommitmentFinalized)
			if err != nil {
				klog.Warningf("Failed to get epoch info, bailing out: %v", err)
			}
			cancel()

			// if we are running for the first time, then we need to set our tracking numbers:
			if c.currentEpoch == 0 {
				c.trackEpoch(epochInfo)
			}

			totalTransactionsTotal.Set(float64(epochInfo.TransactionCount))
			confirmedSlotHeight.Set(float64(epochInfo.AbsoluteSlot))

			// if we get here, then the tracking numbers are set, so this is a "normal" run.
			// start by checking if we have progressed since last run:
			if epochInfo.AbsoluteSlot <= c.slotWatermark {
				klog.Infof("confirmed slot number has not advanced from %v, skipping", c.slotWatermark)
				continue
			}

			if epochInfo.Epoch > c.currentEpoch {
				c.closeCurrentEpoch(ctx, epochInfo)
			}

			// update block production metrics up until the current slot:
			c.fetchAndEmitBlockProduction(ctx, epochInfo.AbsoluteSlot)
		}
	}
}

// trackEpoch takes in a new rpc.EpochInfo and sets the SlotWatcher tracking metrics accordingly,
// and updates the prometheus gauges associated with those metrics.
func (c *SlotWatcher) trackEpoch(epoch *rpc.EpochInfo) {
	firstSlot, lastSlot := getEpochBounds(epoch)
	// if we haven't yet set c.currentEpoch, that (hopefully) means this is the initial setup,
	// and so we can simply store the tracking numbers
	if c.currentEpoch == 0 {
		c.currentEpoch = epoch.Epoch
		c.firstSlot = firstSlot
		c.lastSlot = lastSlot
		// we don't backfill on startup. we set the watermark to current slot minus 1,
		//such that the current slot is the first slot tracked
		c.slotWatermark = epoch.AbsoluteSlot - 1
	} else {
		// if c.currentEpoch is already set, then, just in case, run some checks
		// to make sure that we make sure that we are tracking consistently
		assertf(epoch.Epoch == c.currentEpoch+1, "epoch jumped from %v to %v", c.currentEpoch, epoch.Epoch)
		assertf(
			firstSlot == c.lastSlot+1,
			"first slot %v does not follow from current last slot %v",
			firstSlot,
			c.lastSlot,
		)

		// and also, make sure that we have completed the last epoch:
		assertf(
			c.slotWatermark == c.lastSlot,
			"can't update epoch when watermark %v hasn't reached current last-slot %v",
			c.slotWatermark,
			c.lastSlot,
		)

		// the epoch number is progressing correctly, so we can update our tracking numbers:
		c.currentEpoch = epoch.Epoch
		c.firstSlot = firstSlot
		c.lastSlot = lastSlot
	}

	// emit epoch bounds:
	currentEpochNumber.Set(float64(c.currentEpoch))
	epochFirstSlot.Set(float64(c.firstSlot))
	epochLastSlot.Set(float64(c.lastSlot))
}

// closeCurrentEpoch is called when an epoch change-over happens, and we need to make sure we track the last
// remaining slots in the "current" epoch before we start tracking the new one.
func (c *SlotWatcher) closeCurrentEpoch(ctx context.Context, newEpoch *rpc.EpochInfo) {
	c.fetchAndEmitBlockProduction(ctx, c.lastSlot)
	c.trackEpoch(newEpoch)
}

// checkValidSlotRange makes sure that the slot range we are going to query is within the current epoch we are tracking.
func (c *SlotWatcher) checkValidSlotRange(from, to int64) error {
	if from < c.firstSlot || to > c.lastSlot {
		return fmt.Errorf(
			"start-end slots (%v -> %v) is not contained within current epoch %v range (%v -> %v)",
			from,
			to,
			c.currentEpoch,
			c.firstSlot,
			c.lastSlot,
		)
	}
	return nil
}

// fetchAndEmitBlockProduction fetches block production up to the provided endSlot, emits the prometheus metrics,
// and updates the SlotWatcher.slotWatermark accordingly
func (c *SlotWatcher) fetchAndEmitBlockProduction(ctx context.Context, endSlot int64) {
	// add 1 because GetBlockProduction's range is inclusive, and the watermark is already tracked
	startSlot := c.slotWatermark + 1
	klog.Infof("Fetching block production in [%v -> %v]", startSlot, endSlot)

	// make sure the bounds are contained within the epoch we are currently watching:
	if err := c.checkValidSlotRange(startSlot, endSlot); err != nil {
		klog.Fatalf("invalid slot range: %v", err)
	}

	// fetch block production:
	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	blockProduction, err := c.client.GetBlockProduction(ctx, nil, &startSlot, &endSlot)
	if err != nil {
		klog.Warningf("Failed to get block production, bailing out: %v", err)
	}

	// emit the metrics:
	for address, production := range blockProduction.ByIdentity {
		valid := float64(production.BlocksProduced)
		skipped := float64(production.LeaderSlots - production.BlocksProduced)

		epochStr := fmt.Sprintf("%d", c.currentEpoch)

		leaderSlotsTotal.WithLabelValues("valid", address).Add(valid)
		leaderSlotsTotal.WithLabelValues("skipped", address).Add(skipped)

		if len(c.leaderSlotAddresses) == 0 || slices.Contains(c.leaderSlotAddresses, address) {
			leaderSlotsByEpoch.WithLabelValues("valid", address, epochStr).Add(valid)
			leaderSlotsByEpoch.WithLabelValues("skipped", address, epochStr).Add(skipped)
		}
	}

	klog.Infof("Fetched block production in [%v -> %v]", startSlot, endSlot)
	// update the slot watermark:
	c.slotWatermark = endSlot
}

// getEpochBounds returns the first slot and last slot within an [inclusive] Epoch
func getEpochBounds(info *rpc.EpochInfo) (int64, int64) {
	firstSlot := info.AbsoluteSlot - info.SlotIndex
	return firstSlot, firstSlot + info.SlotsInEpoch - 1
}
