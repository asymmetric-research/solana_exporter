package main

import (
	"github.com/asymmetric-research/solana_exporter/pkg/slog"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	slog.Init()
	code := m.Run()
	os.Exit(code)
}
