package main

import (
	"context"
	"fmt"
	"github.com/certusone/solana_exporter/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
	"time"
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
			Help: "Number of leader slots per leader, grouped by skip status (max confirmation)",
		},
		[]string{"status", "nodekey"})
)

func init() {
	prometheus.MustRegister(totalTransactionsTotal)
	prometheus.MustRegister(confirmedSlotHeight)
	prometheus.MustRegister(currentEpochNumber)
	prometheus.MustRegister(epochFirstSlot)
	prometheus.MustRegister(epochLastSlot)
	prometheus.MustRegister(leaderSlotsTotal)
}

func (c *solanaCollector) WatchSlots() {
	var (
		// Current mapping of relative slot numbers to leader public keys.
		epochSlots map[int64]string
		// Current epoch number corresponding to epochSlots.
		epochNumber int64
		// Last slot number we generated ticks for.
		watermark int64
	)

	ticker := time.NewTicker(slotPacerSchedule)

	for {
		<-ticker.C

		// Get current slot height and epoch info
		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		info, err := c.rpcClient.GetEpochInfo(ctx, rpc.CommitmentMax)
		if err != nil {
			klog.Infof("failed to fetch info info, retrying: %v", err)
			cancel()
			continue
		}
		cancel()

		// Calculate first and last slot in epoch.
		firstSlot := info.AbsoluteSlot - info.SlotIndex
		lastSlot := firstSlot + info.SlotsInEpoch

		totalTransactionsTotal.Set(float64(info.TransactionCount))
		confirmedSlotHeight.Set(float64(info.AbsoluteSlot))
		currentEpochNumber.Set(float64(info.Epoch))
		epochFirstSlot.Set(float64(firstSlot))
		epochLastSlot.Set(float64(lastSlot))

		// Check whether we need to fetch a new leader schedule
		if epochNumber != info.Epoch {
			klog.Infof("new epoch at slot %d: %d (previous: %d)", firstSlot, info.Epoch, epochNumber)

			epochSlots, err = c.fetchLeaderSlots(firstSlot)
			if err != nil {
				klog.Errorf("failed to request leader schedule, retrying: %v", err)
				continue
			}

			klog.V(1).Infof("%d leader slots in epoch %d", len(epochSlots), info.Epoch)

			epochNumber = info.Epoch
			klog.V(1).Infof("we're still in epoch %d, not fetching leader schedule", info.Epoch)

			// Reset watermark to current offset on new epoch (we do not backfill slots we missed at startup)
			watermark = info.SlotIndex
		} else if watermark == info.SlotIndex {
			klog.Infof("slot has not advanced at %d, skipping", info.AbsoluteSlot)
			continue
		}

		klog.Infof("confirmed slot %d (offset %d, +%d), epoch %d (from slot %d to %d, %d remaining)",
			info.AbsoluteSlot, info.SlotIndex, info.SlotIndex-watermark, info.Epoch, firstSlot, lastSlot, lastSlot-info.AbsoluteSlot)

		// Get list of confirmed blocks since the last request. This is totally undocumented, but the result won't
		// contain missed blocks, allowing us to figure out block production success rate.
		rangeStart := firstSlot + watermark
		rangeEnd := firstSlot + info.SlotIndex - 1

		ctx, cancel = context.WithTimeout(context.Background(), httpTimeout)
		cfm, err := c.rpcClient.GetConfirmedBlocks(ctx, rangeStart, rangeEnd)
		if err != nil {
			klog.Errorf("failed to request confirmed blocks at %d, retrying: %v", watermark, err)
			cancel()
			continue
		}
		cancel()

		klog.V(1).Infof("confirmed blocks: %d -> %d: %v", rangeStart, rangeEnd, cfm)

		// Figure out leaders for each block in range
		for i := watermark; i < info.SlotIndex; i++ {
			leader, ok := epochSlots[i]
			abs := firstSlot + i
			if !ok {
				// This cannot happen with a well-behaved node and is a programming error in either Solana or the exporter.
				klog.Fatalf("slot %d (offset %d) missing from epoch %d leader schedule",
					abs, i, info.Epoch)
			}

			// Check if block was included in getConfirmedBlocks output, otherwise, it was skipped.
			var present bool
			for _, s := range cfm {
				if abs == s {
					present = true
				}
			}

			var skipped string
			var label string
			if present {
				skipped = "(valid)"
				label = "valid"
			} else {
				skipped = "(SKIPPED)"
				label = "skipped"
			}

			leaderSlotsTotal.With(prometheus.Labels{"status": label, "nodekey": leader}).Add(1)
			klog.V(1).Infof("slot %d (offset %d) with leader %s %s", abs, i, leader, skipped)
		}

		watermark = info.SlotIndex
	}
}

func (c *solanaCollector) fetchLeaderSlots(epochSlot int64) (map[int64]string, error) {
	sch, err := c.rpcClient.GetLeaderSchedule(context.Background(), epochSlot)
	if err != nil {
		return nil, fmt.Errorf("failed to get leader schedule: %w", err)
	}

	slots := make(map[int64]string)

	for pk, sch := range sch {
		for _, i := range sch {
			slots[int64(i)] = pk
		}
	}

	return slots, err
}
