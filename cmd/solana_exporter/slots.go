package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

const (
	slotPacerSchedule = 1 * time.Second
)

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

func init() {
	prometheus.MustRegister(totalTransactionsTotal)
	prometheus.MustRegister(confirmedSlotHeight)
	prometheus.MustRegister(currentEpochNumber)
	prometheus.MustRegister(epochFirstSlot)
	prometheus.MustRegister(epochLastSlot)
	prometheus.MustRegister(leaderSlotsTotal)
	prometheus.MustRegister(leaderSlotsByEpoch)
}

func (c *solanaCollector) WatchSlots(ctx context.Context) {
	// Get current slot height and epoch info
	ctx_, cancel := context.WithTimeout(ctx, httpTimeout)
	epochInfo, err := c.rpcClient.GetEpochInfo(ctx_, rpc.CommitmentFinalized)
	if err != nil {
		klog.Fatalf("failed to fetch epoch info, bailing out: %v", err)
	}
	cancel()

	// watermark is the last slot number we generated ticks for. Set it to the current offset on startup (
	// we do not backfill slots we missed at startup)
	watermark := epochInfo.AbsoluteSlot
	currentEpoch, firstSlot, lastSlot := getEpochBounds(epochInfo)
	currentEpochNumber.Set(float64(currentEpoch))
	epochFirstSlot.Set(float64(firstSlot))
	epochLastSlot.Set(float64(lastSlot))
	totalTransactionsTotal.Set(float64(epochInfo.TransactionCount))
	confirmedSlotHeight.Set(float64(epochInfo.AbsoluteSlot))

	klog.Infof("Starting at slot %d in epoch %d (%d-%d)", firstSlot, currentEpoch, firstSlot, lastSlot)
	_, err = c.updateCounters(currentEpoch, watermark, &lastSlot)
	if err != nil {
		klog.Error(err)
	}
	ticker := time.NewTicker(c.slotPace)

	for {
		select {
		case <-ctx.Done():
			klog.Infof("Stopping WatchSlots() at slot %v", watermark)
			return

		default:
			<-ticker.C

			// Get current slot height and epoch info
			ctx_, cancel := context.WithTimeout(context.Background(), httpTimeout)
			info, err := c.rpcClient.GetEpochInfo(ctx_, rpc.CommitmentFinalized)
			if err != nil {
				klog.Warningf("failed to fetch epoch info, retrying: %v", err)
				cancel()
				continue
			}
			cancel()

			if watermark == info.AbsoluteSlot {
				klog.V(2).Infof("slot has not advanced at %d, skipping", info.AbsoluteSlot)
				continue
			}

			if currentEpoch != info.Epoch {
				klog.Infof(
					"changing epoch from %d to %d. Watermark: %d, lastSlot: %d",
					currentEpoch,
					info.Epoch,
					watermark,
					lastSlot,
				)

				last, err := c.updateCounters(currentEpoch, watermark, &lastSlot)
				if err != nil {
					klog.Error(err)
					continue
				}

				klog.Infof(
					"counters updated to slot %d (+%d), epoch %d (slots %d-%d, %d remaining)",
					last,
					last-watermark,
					currentEpoch,
					firstSlot,
					lastSlot,
					lastSlot-last,
				)

				watermark = last
				currentEpoch, firstSlot, lastSlot = getEpochBounds(info)

				currentEpochNumber.Set(float64(currentEpoch))
				epochFirstSlot.Set(float64(firstSlot))
				epochLastSlot.Set(float64(lastSlot))
			}

			totalTransactionsTotal.Set(float64(info.TransactionCount))
			confirmedSlotHeight.Set(float64(info.AbsoluteSlot))

			last, err := c.updateCounters(currentEpoch, watermark, nil)
			if err != nil {
				klog.Info(err)
				continue
			}

			klog.Infof(
				"counters updated to slot %d (offset %d, +%d), epoch %d (slots %d-%d, %d remaining)",
				last,
				info.SlotIndex,
				last-watermark,
				currentEpoch,
				firstSlot,
				lastSlot,
				lastSlot-last,
			)

			watermark = last
		}
	}
}

// getEpochBounds returns the epoch, first slot and last slot given an EpochInfo struct
func getEpochBounds(info *rpc.EpochInfo) (int64, int64, int64) {
	firstSlot := info.AbsoluteSlot - info.SlotIndex
	lastSlot := firstSlot + info.SlotsInEpoch

	return info.Epoch, firstSlot, lastSlot
}

func (c *solanaCollector) updateCounters(epoch, firstSlot int64, lastSlotOpt *int64) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	var lastSlot int64
	var err error

	if lastSlotOpt == nil {
		lastSlot, err = c.rpcClient.GetSlot(ctx)

		if err != nil {
			return 0, fmt.Errorf("error while getting the last slot: %v", err)
		}
		klog.V(2).Infof("Setting lastSlot to %d", lastSlot)
	} else {
		lastSlot = *lastSlotOpt
		klog.Infof("Got lastSlot: %d", lastSlot)
	}

	if firstSlot > lastSlot {
		return 0, fmt.Errorf(
			"in epoch %d, firstSlot (%d) > lastSlot (%d), this should not happen, not updating",
			epoch,
			firstSlot,
			lastSlot,
		)
	}

	ctx, cancel = context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	blockProduction, err := c.rpcClient.GetBlockProduction(ctx, nil, &firstSlot, &lastSlot)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch block production, retrying: %v", err)
	}

	for identity, production := range blockProduction.ByIdentity {
		valid := float64(production.BlocksProduced)
		skipped := float64(production.LeaderSlots - production.BlocksProduced)

		epochStr := fmt.Sprintf("%d", epoch)

		leaderSlotsTotal.WithLabelValues("valid", identity).Add(valid)
		leaderSlotsTotal.WithLabelValues("skipped", identity).Add(skipped)

		if len(c.leaderSlotAddresses) == 0 || slices.Contains(c.leaderSlotAddresses, identity) {
			leaderSlotsByEpoch.WithLabelValues("valid", identity, epochStr).Add(valid)
			leaderSlotsByEpoch.WithLabelValues("skipped", identity, epochStr).Add(skipped)
		}

		klog.V(1).Infof(
			"Epoch %s, slots %d-%d, node %s: Added %d valid and %d skipped slots",
			epochStr,
			firstSlot,
			lastSlot,
			identity,
			production.BlocksProduced,
			production.LeaderSlots-production.BlocksProduced,
		)
	}

	return lastSlot, nil
}
