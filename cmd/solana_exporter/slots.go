package main

import (
	"context"
	"fmt"
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
			Help: "(DEPRECATED) Number of leader slots per leader, grouped by skip status",
		},
		[]string{"status", "nodekey"})

	leaderSlotsByEpoch = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "solana_leader_slots_by_epoch",
			Help: "Number of leader slots per leader, grouped by skip status and epoch",
		},
		[]string{"status", "nodekey", "epoch"})
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

func (c *solanaCollector) WatchSlots() {
	// Get current slot height and epoch info
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	info, err := c.rpcClient.GetEpochInfo(ctx, rpc.CommitmentMax)
	if err != nil {
		klog.Infof("failed to fetch epoch info, bailing out: %v", err)
		os.Exit(1)
	}
	cancel()

	// watermark is the last slot number we generated ticks for. Set it to the current offset on startup (we do not backfill slots we missed at startup)
	watermark := info.AbsoluteSlot
	currentEpoch := info.Epoch
	firstSlot := info.AbsoluteSlot - info.SlotIndex
	lastSlot := firstSlot + info.SlotsInEpoch

	ticker := time.NewTicker(slotPacerSchedule)

	for {
		<-ticker.C

		// Get current slot height and epoch info
		ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
		info, err := c.rpcClient.GetEpochInfo(ctx, rpc.CommitmentMax)
		if err != nil {
			klog.Infof("failed to fetch epoch info, retrying: %v", err)
			cancel()
			continue
		}
		cancel()

		if currentEpoch != info.Epoch {
			last, err := updateCounters(c.rpcClient, currentEpoch, watermark, &lastSlot)
			if err != nil {
				klog.Info(err)
				continue
			}
			watermark = last
		}

		currentEpoch = info.Epoch
		// Calculate first and last slot in epoch.
		firstSlot = info.AbsoluteSlot - info.SlotIndex
		lastSlot := firstSlot + info.SlotsInEpoch

		totalTransactionsTotal.Set(float64(info.TransactionCount))
		confirmedSlotHeight.Set(float64(info.AbsoluteSlot))
		currentEpochNumber.Set(float64(info.Epoch))
		epochFirstSlot.Set(float64(firstSlot))
		epochLastSlot.Set(float64(lastSlot))

		if watermark == info.AbsoluteSlot {
			klog.Infof("slot has not advanced at %d, skipping", info.AbsoluteSlot)
			continue
		}

		klog.Infof("confirmed slot %d (offset %d, +%d), epoch %d (from slot %d to %d, %d remaining)",
			info.AbsoluteSlot, info.SlotIndex, info.AbsoluteSlot-watermark, info.Epoch, firstSlot, lastSlot, lastSlot-info.AbsoluteSlot)

		last, err := updateCounters(c.rpcClient, currentEpoch, watermark, nil)
		if err != nil {
			klog.Info(err)
			continue
		}
		watermark = last
	}
}

func updateCounters(c *rpc.RPCClient, epoch, firstSlot int64, lastSlotOpt *int64) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	var lastSlot int64
	var err error

	if lastSlotOpt == nil {
		klog.V(2).Info("LastSlot is nil, getting last published slot")
		lastSlot, err = c.GetSlot(ctx)

		if err != nil {
			cancel()
			return 0, fmt.Errorf("Error while getting the last slot: %v", err)
		}
		cancel()
	} else {
		lastSlot = *lastSlotOpt
	}
	klog.V(2).Infof("LastSlot is %d", lastSlot)

	if firstSlot > lastSlot {
		return 0, fmt.Errorf(
			"In epoch %d, firstSlot (%d) > lastSlot (%d). This should not happen. Not updating.",
			epoch,
			firstSlot,
			lastSlot,
		)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	blockProduction, err := c.GetBlockProduction(ctx, &firstSlot, &lastSlot)
	if err != nil {
		cancel()
		return 0, fmt.Errorf("failed to fetch block production, retrying: %v", err)
	}
	cancel()

	for host, prod := range blockProduction.Hosts {
		valid := float64(prod.BlocksProduced)
		skipped := float64(prod.LeaderSlots - prod.BlocksProduced)

		epochStr := fmt.Sprintf("%d", epoch)

		leaderSlotsTotal.WithLabelValues("valid", host).Add(valid)
		leaderSlotsTotal.WithLabelValues("skipped", host).Add(skipped)

		leaderSlotsByEpoch.WithLabelValues("valid", host, epochStr).Add(valid)
		leaderSlotsByEpoch.WithLabelValues("skipped", host, epochStr).Add(skipped)

		klog.V(1).Infof(
			"Epoch %s, slots %d-%d, node %s: Added %d valid and %d skipped slots",
			epochStr,
			firstSlot,
			lastSlot,
			host,
			prod.BlocksProduced,
			prod.LeaderSlots-prod.BlocksProduced,
		)
	}

	return lastSlot, nil
}
