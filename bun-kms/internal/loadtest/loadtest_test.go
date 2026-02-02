package loadtest

import (
	"context"
	"encoding/base64"
	"log"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kartikbazzad/bunbase/bun-kms/internal/api"
	"github.com/kartikbazzad/bunbase/bun-kms/internal/core"
)

func TestLoadTest_InProcess_Mixed(t *testing.T) {
	logger := log.New(nil, "", 0)
	masterKey, _ := core.ParseMasterKey("base64:" + base64.StdEncoding.EncodeToString(core.TestMasterKey))
	vault := core.NewVault()
	secrets, err := core.NewSecretStore(masterKey)
	if err != nil {
		t.Fatalf("NewSecretStore: %v", err)
	}
	server := api.NewServer(vault, secrets, logger, nil)
	srv := httptest.NewServer(server.Handler())
	defer srv.Close()

	cfg := DefaultConfig(srv.URL)
	cfg.Duration = 2 * time.Second
	cfg.NumClients = 5
	cfg.PayloadSize = 64
	cfg.Workload = WorkloadMixed

	ctx := context.Background()
	report, err := Run(ctx, cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.TotalOps == 0 {
		t.Error("expected total_ops > 0")
	}
	if report.OpsPerSec <= 0 && report.TotalOps > 0 {
		t.Error("expected ops_per_sec > 0")
	}
	t.Logf("total_ops=%d errors=%d ops_per_sec=%.2f p50=%v p95=%v p99=%v",
		report.TotalOps, report.Errors, report.OpsPerSec,
		report.P50Latency, report.P95Latency, report.P99Latency)
}

func TestLoadTest_InProcess_Encrypt(t *testing.T) {
	logger := log.New(nil, "", 0)
	masterKey, _ := core.ParseMasterKey("base64:" + base64.StdEncoding.EncodeToString(core.TestMasterKey))
	vault := core.NewVault()
	secrets, _ := core.NewSecretStore(masterKey)
	server := api.NewServer(vault, secrets, logger, nil)
	srv := httptest.NewServer(server.Handler())
	defer srv.Close()

	cfg := DefaultConfig(srv.URL)
	cfg.Duration = 1 * time.Second
	cfg.NumClients = 3
	cfg.Workload = WorkloadEncrypt

	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.TotalOps == 0 {
		t.Error("expected total_ops > 0")
	}
	t.Logf("encrypt total_ops=%d ops_per_sec=%.2f", report.TotalOps, report.OpsPerSec)
}

func TestLoadTest_InProcess_Secrets(t *testing.T) {
	logger := log.New(nil, "", 0)
	masterKey, _ := core.ParseMasterKey("base64:" + base64.StdEncoding.EncodeToString(core.TestMasterKey))
	vault := core.NewVault()
	secrets, _ := core.NewSecretStore(masterKey)
	server := api.NewServer(vault, secrets, logger, nil)
	srv := httptest.NewServer(server.Handler())
	defer srv.Close()

	cfg := DefaultConfig(srv.URL)
	cfg.Duration = 1 * time.Second
	cfg.NumClients = 3
	cfg.Workload = WorkloadSecrets

	report, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.TotalOps == 0 {
		t.Error("expected total_ops > 0")
	}
	t.Logf("secrets total_ops=%d ops_per_sec=%.2f", report.TotalOps, report.OpsPerSec)
}

func TestLoadTest_Client_EncryptDecrypt(t *testing.T) {
	logger := log.New(nil, "", 0)
	masterKey, _ := core.ParseMasterKey("base64:" + base64.StdEncoding.EncodeToString(core.TestMasterKey))
	vault := core.NewVault()
	vault.CreateKey("test-key", core.KeyTypeAES256)
	secrets, _ := core.NewSecretStore(masterKey)
	server := api.NewServer(vault, secrets, logger, nil)
	srv := httptest.NewServer(server.Handler())
	defer srv.Close()

	client := NewClient(srv.URL, "")
	plaintext := []byte("hello")
	ciphertext, err := client.Encrypt("test-key", plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	decrypted, err := client.Decrypt("test-key", ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func BenchmarkLoadTest_Mixed(b *testing.B) {
	logger := log.New(nil, "", 0)
	masterKey, _ := core.ParseMasterKey("base64:" + base64.StdEncoding.EncodeToString(core.TestMasterKey))
	vault := core.NewVault()
	secrets, _ := core.NewSecretStore(masterKey)
	server := api.NewServer(vault, secrets, logger, nil)
	srv := httptest.NewServer(server.Handler())
	defer srv.Close()

	cfg := DefaultConfig(srv.URL)
	cfg.Duration = 1 * time.Second
	cfg.NumClients = 10
	cfg.Workload = WorkloadMixed

	report, err := Run(context.Background(), cfg)
	if err != nil {
		b.Fatalf("Run: %v", err)
	}
	b.ReportMetric(report.OpsPerSec, "ops/sec")
	b.ReportMetric(float64(report.P50Latency.Microseconds()), "p50_us")
}
