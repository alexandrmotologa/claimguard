package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/registry"
	"github.com/alexandrmotologa/claimguard/internal/rewriter"
	"github.com/alexandrmotologa/claimguard/internal/server"
	"github.com/alexandrmotologa/claimguard/internal/telemetry"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

func setupTestProxy(t *testing.T) (*ProxyHandler, *telemetry.AuditLogger, *telemetry.MetricsTracker) {
	store, err := registry.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	seedPath := filepath.Join("..", "..", "examples", "ground_truth.json")
	if _, err := store.LoadFromFile(seedPath); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Policy.DefaultAction = "rewrite"

	audit, err := telemetry.NewAuditLogger(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	metrics := telemetry.NewMetricsTracker()
	broadcaster := server.NewBroadcaster()

	ext := extractor.NewExtractor(store)
	ver := verifier.NewEngine(store, cfg.Policy)
	rew := rewriter.NewRewriter(cfg.Policy)
	sim := NewSimulator(1 * time.Millisecond)

	handler := NewProxyHandler(cfg, sim, ext, ver, rew, audit, metrics, broadcaster)
	return handler, audit, metrics
}

func TestProxyStreamingInterception(t *testing.T) {
	handler, audit, metrics := setupTestProxy(t)

	// User asks for return policy (simulator will try to claim 60 days)
	reqPayload := ChatCompletionRequest{
		Model:  "gpt-4o",
		Stream: true,
		Messages: []ChatMessage{
			{Role: "user", Content: "What is your refund policy window?"},
		},
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	rawBody := w.Body.String()
	if !strings.Contains(rawBody, "data: [DONE]") {
		t.Fatalf("expected data: [DONE] in stream output, got:\n%s", rawBody)
	}

	// Verify that "14 days" or "14-day" replaced the hallucinated 60 days
	if !strings.Contains(rawBody, "14-day") && !strings.Contains(rawBody, "14 days") {
		t.Fatalf("expected rewritten 14-day policy in stream, got:\n%s", rawBody)
	}

	// Verify metrics and audit record
	snap := metrics.Snapshot()
	if snap.TotalContradictions == 0 {
		t.Fatalf("expected recorded contradictions, got 0")
	}

	events, count, err := audit.ListEvents(10, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 || len(events) == 0 {
		t.Fatalf("expected audit events in database, got %d", count)
	}
}

func TestProxyNonStreamingInterception(t *testing.T) {
	handler, _, _ := setupTestProxy(t)

	reqPayload := ChatCompletionRequest{
		Model:  "gpt-4o",
		Stream: false,
		Messages: []ChatMessage{
			{Role: "user", Content: "How much does the Enterprise plan cost per month?"},
		},
	}
	body, _ := json.Marshal(reqPayload)

	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var completionResp NonStreamingResponse
	if err := json.NewDecoder(w.Body).Decode(&completionResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	content := completionResp.Choices[0].Message.Content
	// Should have rewritten $99/month to $499/month
	if !strings.Contains(content, "$499/month") {
		t.Fatalf("expected rewritten $499/month in non-streaming response, got:\n%s", content)
	}
}

func TestSimulatorOffline(t *testing.T) {
	sim := NewSimulator(0)
	resp, err := sim.Complete(context.Background(), &ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []ChatMessage{
			{Role: "user", Content: "Tell me about personal loan APR"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("empty choices")
	}
	if !strings.Contains(resp.Choices[0].Message.Content, "29.9%") {
		t.Fatalf("expected 29.9%% in loan APR simulation, got: %s", resp.Choices[0].Message.Content)
	}
}
