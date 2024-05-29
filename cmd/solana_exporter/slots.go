package main

import (
	"context"
	"os"
	"time"

	"github.com/certusone/solana_exporter/pkg/rpc"
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
		// Last slot number we generated ticks for.
		watermark int64
	)

	// Get current slot height and epoch info
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	info, err := c.rpcClient.GetEpochInfo(ctx, rpc.CommitmentMax)
	if err != nil {
		klog.Infof("failed to fetch epoch info, bailing out: %v", err)
		os.Exit(1)
	}
	cancel()

	// Set watermark to current offset on startup (we do not backfill slots we missed at startup)
	watermark = info.AbsoluteSlot

	ticker := time.NewTicker(slotPacerSchedule)

	for {
		<-ticker.C

		// Get current slot height and epoch info
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		info, err := c.rpcClient.GetEpochInfo(ctx, rpc.CommitmentMax)
		if err != nil {
			klog.Infof("failed to fetch epoch info, retrying: %v", err)
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

		if watermark == info.SlotIndex {
			klog.Infof("slot has not advanced at %d, skipping", info.AbsoluteSlot)
			continue
		}

		klog.Infof("confirmed slot %d (offset %d, +%d), epoch %d (from slot %d to %d, %d remaining)",
			info.AbsoluteSlot, info.SlotIndex, info.AbsoluteSlot-watermark, info.Epoch, firstSlot, lastSlot, lastSlot-info.AbsoluteSlot)

		first := watermark + 1
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		blockProduction, err := c.rpcClient.GetBlockProduction(ctx, &first, nil)
		if err != nil {
			klog.Infof("failed to fetch block production, retrying: %v", err)
			cancel()
			continue
		}

		for host, prod := range blockProduction.Hosts {
			leaderSlotsTotal.
				With(prometheus.Labels{"status": "valid", "nodekey": host}).
				Add(float64(prod.BlocksProduced))
			leaderSlotsTotal.
				With(prometheus.Labels{"status": "skipped", "nodekey": host}).
				Add(float64(prod.LeaderSlots - prod.BlocksProduced))
			klog.V(1).Infof(
				"Slot %d, node %s: Added %d valid and %d skipped slots",
				blockProduction.LastSlot,
				host,
				prod.BlocksProduced,
				prod.LeaderSlots-prod.BlocksProduced,
			)
		}

		watermark = blockProduction.LastSlot
	}
}
