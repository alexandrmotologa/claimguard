package benchmark

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/proxy"
	"github.com/alexandrmotologa/claimguard/internal/registry"
	"github.com/alexandrmotologa/claimguard/internal/rewriter"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

func TestBenchmarkRunner(t *testing.T) {
	store, err := registry.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seedPath := filepath.Join("..", "..", "examples", "ground_truth.json")
	if _, err := store.LoadFromFile(seedPath); err != nil {
		t.Fatal(err)
	}

	scenarios := []Scenario{
		{
			ID:           "test-1",
			Domain:       "commerce",
			Prompt:       "Can I return items within 60 days?",
			TargetEntity: "policy.refund",
		},
		{
			ID:           "test-2",
			Domain:       "pricing",
			Prompt:       "How much is the enterprise plan per month?",
			TargetEntity: "tier.enterprise",
		},
	}

	cfg := config.DefaultConfig()
	ext := extractor.NewExtractor(store)
	ver := verifier.NewEngine(store, cfg.Policy)
	rew := rewriter.NewRewriter(cfg.Policy)
	sim := proxy.NewSimulator(0)

	runner := NewRunner(store, ext, ver, rew, sim)
	report, err := runner.RunSuite(context.Background(), "Unit Test Benchmark", scenarios)
	if err != nil {
		t.Fatal(err)
	}

	if report.TotalScenarios != 2 {
		t.Fatalf("expected 2 scenarios, got %d", report.TotalScenarios)
	}
	if report.PassedScenarios != 2 {
		t.Fatalf("expected 2 passed, got %d", report.PassedScenarios)
	}
	if report.ComplianceAccuracyScore != 100.0 {
		t.Fatalf("expected 100%%, got %.1f%%", report.ComplianceAccuracyScore)
	}
	if report.CertificationHash == "" {
		t.Fatal("expected non-empty certification hash")
	}
}
