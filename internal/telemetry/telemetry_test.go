package telemetry

import (
	"testing"
	"time"
)

func TestAuditLoggerAndIntegrity(t *testing.T) {
	logger, err := NewAuditLogger(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	event := AuditEvent{
		ID:                 "evt-1",
		Timestamp:          time.Now().UTC(),
		RequestID:          "req-123",
		PromptPreview:      "What is the refund window?",
		OriginalSentence:   "Our policy is 60 days.",
		CorrectedSentence:  "Our policy is 14 days.",
		EntityID:           "policy.refund",
		Attribute:          "window_days",
		AssertedValue:      "60 days",
		AuthoritativeValue: "14 days",
		PolicyAction:       "rewrite",
		Status:             "CONTRADICTED",
		LatencyMs:          2.4,
	}

	if err := logger.RecordEvent(event); err != nil {
		t.Fatalf("record event: %v", err)
	}

	events, count, err := logger.ListEvents(10, 0, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(events) != 1 {
		t.Fatalf("expected 1 event, got count %d, len %d", count, len(events))
	}

	valid, invalid, err := logger.VerifyIntegrity()
	if err != nil {
		t.Fatal(err)
	}
	if valid != 1 || invalid != 0 {
		t.Fatalf("integrity check failed: valid %d, invalid %d", valid, invalid)
	}

	csvBytes, err := logger.ExportEvents("csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(csvBytes) == 0 {
		t.Fatal("empty csv export")
	}
}

func TestMetricsTracker(t *testing.T) {
	tracker := NewMetricsTracker()
	tracker.RecordRequest(120)
	tracker.RecordSentence()
	tracker.RecordSentence()
	tracker.RecordClaimVerified()
	tracker.RecordClaimVerified()
	tracker.RecordContradiction("rewrite", "policy.refund", 4.5)

	snap := tracker.Snapshot()
	if snap.TotalRequests != 1 {
		t.Fatalf("expected 1 request, got %d", snap.TotalRequests)
	}
	if snap.TotalTokens != 120 {
		t.Fatalf("expected 120 tokens, got %d", snap.TotalTokens)
	}
	if snap.TotalSentences != 2 {
		t.Fatalf("expected 2 sentences, got %d", snap.TotalSentences)
	}
	if snap.TotalClaimsVerified != 2 {
		t.Fatalf("expected 2 claims, got %d", snap.TotalClaimsVerified)
	}
	if snap.TotalContradictions != 1 {
		t.Fatalf("expected 1 contradiction, got %d", snap.TotalContradictions)
	}
	if snap.InterventionsByAction["rewrite"] != 1 {
		t.Fatalf("expected 1 rewrite, got %d", snap.InterventionsByAction["rewrite"])
	}
	if snap.ViolationsByEntity["policy.refund"] != 1 {
		t.Fatalf("expected 1 violation for policy.refund, got %d", snap.ViolationsByEntity["policy.refund"])
	}
}
