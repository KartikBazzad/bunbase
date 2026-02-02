package loadtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bunder/internal/config"
	"github.com/kartikbazzad/bunbase/bunder/internal/server"
	"github.com/kartikbazzad/bunbase/bunder/pkg/client"
)

// TestLoadTest_InProcess starts a Bunder server in-process and runs a short load test.
func TestLoadTest_InProcess(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.DataPath = dir
	cfg.ListenAddr = "127.0.0.1:0" // random port
	cfg.HTTPAddr = ""              // disable HTTP for this test
	cfg.BufferPoolSize = 500
	cfg.Shards = 16
	// Do not call ParseFlags() in tests to avoid flag redefinition

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Close()

	addr := srv.Addr()
	if addr == "" {
		t.Fatal("server Addr() is empty")
	}

	ctx := context.Background()
	cfgLoad := DefaultConfig(addr)
	cfgLoad.Duration = 2 * time.Second
	cfgLoad.NumClients = 10
	cfgLoad.KeySpace = 1000
	cfgLoad.ValueSize = 32
	cfgLoad.Workload = WorkloadMixed

	report, err := Run(ctx, cfgLoad)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if report.TotalOps == 0 {
		t.Error("expected TotalOps > 0")
	}
	if report.Errors > 0 {
		t.Logf("load test had %d errors (may be acceptable)", report.Errors)
	}
	t.Logf("load test: ops=%d errors=%d duration=%v ops/sec=%.0f P50=%v P95=%v P99=%v",
		report.TotalOps, report.Errors, report.Duration,
		report.OpsPerSec, report.P50Latency, report.P95Latency, report.P99Latency)
}

// TestLoadTest_SetOnly runs a short set-only load test against an in-process server.
func TestLoadTest_SetOnly(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.DataPath = dir
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.HTTPAddr = ""
	cfg.BufferPoolSize = 500
	cfg.Shards = 16

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Close()

	ctx := context.Background()
	cfgLoad := DefaultConfig(srv.Addr())
	cfgLoad.Duration = 1 * time.Second
	cfgLoad.NumClients = 5
	cfgLoad.Workload = WorkloadSet

	report, err := Run(ctx, cfgLoad)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.TotalOps == 0 {
		t.Error("expected TotalOps > 0")
	}
	t.Logf("set-only: ops=%d ops/sec=%.0f", report.TotalOps, report.OpsPerSec)
}

// TestLoadTest_GetOnly runs a short get-only load test; pre-populates keys first.
func TestLoadTest_GetOnly(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.DataPath = dir
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.HTTPAddr = ""
	cfg.BufferPoolSize = 500
	cfg.Shards = 16

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Close()

	addr := srv.Addr()
	ctx := context.Background()
	c, err := client.Connect(ctx, client.DefaultOptions(addr))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Pre-populate 100 keys so GET has something to read
	for i := 0; i < 100; i++ {
		_ = c.Set(ctx, fmt.Sprintf("pre%d", i), []byte("value"))
	}
	c.Close()

	cfgLoad := DefaultConfig(addr)
	cfgLoad.Duration = 1 * time.Second
	cfgLoad.NumClients = 5
	cfgLoad.KeySpace = 100
	cfgLoad.Workload = WorkloadGet

	report, err := Run(ctx, cfgLoad)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.TotalOps == 0 {
		t.Error("expected TotalOps > 0")
	}
	t.Logf("get-only: ops=%d ops/sec=%.0f", report.TotalOps, report.OpsPerSec)
}

// BenchmarkLoadTest_Mixed runs a 1s mixed workload with 10 clients (for CI/quick benchmark).
func BenchmarkLoadTest_Mixed(b *testing.B) {
	b.StopTimer()
	dir := b.TempDir()
	cfg := config.Default()
	cfg.DataPath = dir
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.HTTPAddr = ""
	cfg.BufferPoolSize = 500
	cfg.Shards = 16

	srv, err := server.NewServer(cfg)
	if err != nil {
		b.Fatalf("NewServer: %v", err)
	}
	if err := srv.Start(); err != nil {
		b.Fatalf("Start: %v", err)
	}
	defer srv.Close()

	ctx := context.Background()
	cfgLoad := DefaultConfig(srv.Addr())
	cfgLoad.Duration = 1 * time.Second
	cfgLoad.NumClients = 10
	cfgLoad.KeySpace = 1000
	cfgLoad.ValueSize = 64
	cfgLoad.Workload = WorkloadMixed

	b.StartTimer()
	report, err := Run(ctx, cfgLoad)
	b.StopTimer()
	if err != nil {
		b.Fatalf("Run: %v", err)
	}
	b.ReportMetric(report.OpsPerSec, "ops/sec")
	b.ReportMetric(float64(report.P50Latency.Microseconds()), "P50_us")
	b.ReportMetric(float64(report.P99Latency.Microseconds()), "P99_us")
}
