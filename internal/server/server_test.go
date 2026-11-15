package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/registry"
	"github.com/alexandrmotologa/claimguard/internal/rewriter"
	"github.com/alexandrmotologa/claimguard/internal/telemetry"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

func setupTestServer(t *testing.T) (*APIServer, *http.ServeMux) {
	store, err := registry.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	seedPath := filepath.Join("..", "..", "examples", "ground_truth.json")
	if _, err := store.LoadFromFile(seedPath); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	ver := verifier.NewEngine(store, cfg.Policy)
	rew := rewriter.NewRewriter(cfg.Policy)
	ext := extractor.NewExtractor(store)

	audit, err := telemetry.NewAuditLogger(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	metrics := telemetry.NewMetricsTracker()
	broadcaster := NewBroadcaster()

	api := NewAPIServer(store, ver, rew, ext, audit, metrics, broadcaster, cfg)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	return api, mux
}

func TestAPIHealthAndMetrics(t *testing.T) {
	_, mux := setupTestServer(t)

	// Health
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Metrics
	req2 := httptest.NewRequest("GET", "/api/metrics", nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
}

func TestAPIDryRun(t *testing.T) {
	_, mux := setupTestServer(t)

	dryRunReq := DryRunRequest{
		Sentence:      "We offer a 60-day return policy for all items.",
		PromptContext: "What is your refund policy?",
	}
	body, _ := json.Marshal(dryRunReq)

	req := httptest.NewRequest("POST", "/api/verify/dry-run", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DryRunResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if len(resp.Claims) == 0 {
		t.Fatal("expected extracted claims")
	}
	if resp.PrimaryAction != "rewrite" {
		t.Fatalf("expected rewrite action, got %s", resp.PrimaryAction)
	}
	if resp.TransformedSentence != "We offer a 14-day return policy for all items." {
		t.Fatalf("expected rewritten sentence, got %q", resp.TransformedSentence)
	}
}
