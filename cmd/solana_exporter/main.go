package main

import (
	"context"
	"github.com/asymmetric-research/solana_exporter/pkg/rpc"
	"github.com/asymmetric-research/solana_exporter/pkg/slog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

func main() {
	slog.Init()
	logger := slog.Get()
	ctx := context.Background()

	config, err := NewExporterConfigFromCLI(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	if config.ComprehensiveSlotTracking {
		logger.Warn(
			"Comprehensive slot tracking will lead to potentially thousands of new " +
				"Prometheus metrics being created every epoch.",
		)
	}

	client := rpc.NewRPCClient(config.RpcUrl, config.HttpTimeout)
	collector := NewSolanaCollector(client, config)
	slotWatcher := NewSlotWatcher(client, config)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go slotWatcher.WatchSlots(ctx)

	prometheus.MustRegister(collector)
	http.Handle("/metrics", promhttp.Handler())

	logger.Infof("listening on %s", config.ListenAddress)
	logger.Fatal(http.ListenAndServe(config.ListenAddress, nil))
}
