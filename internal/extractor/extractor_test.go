package extractor

import (
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/claimguard/internal/registry"
)

func TestSplitSentences(t *testing.T) {
	text := "We charge $49.99 for the service. Dr. Smith approved this! Can you return it within 14 days? Yes."
	sentences, rem := SplitSentences(text, true)

	if rem != "" {
		t.Fatalf("expected empty remainder, got %q", rem)
	}
	if len(sentences) != 4 {
		t.Fatalf("expected 4 sentences, got %d: %+v", len(sentences), sentences)
	}

	if sentences[0].Text != "We charge $49.99 for the service." {
		t.Fatalf("sentence 0 mismatch: %q", sentences[0].Text)
	}
	if sentences[1].Text != "Dr. Smith approved this!" {
		t.Fatalf("sentence 1 mismatch: %q", sentences[1].Text)
	}
	if sentences[2].Text != "Can you return it within 14 days?" {
		t.Fatalf("sentence 2 mismatch: %q", sentences[2].Text)
	}
	if sentences[3].Text != "Yes." {
		t.Fatalf("sentence 3 mismatch: %q", sentences[3].Text)
	}
}

func TestSplitSentencesStreamingIncomplete(t *testing.T) {
	chunk1 := "Our refund policy allows you to return any item within 60 days if you"
	sentences, rem := SplitSentences(chunk1, false)

	if len(sentences) != 0 {
		t.Fatalf("expected 0 completed sentences in incomplete stream, got %d", len(sentences))
	}
	if rem != chunk1 {
		t.Fatalf("expected remainder %q, got %q", chunk1, rem)
	}

	chunk2 := rem + " are not satisfied. Next sentence begins"
	sentences2, rem2 := SplitSentences(chunk2, false)

	if len(sentences2) != 1 {
		t.Fatalf("expected 1 completed sentence, got %d", len(sentences2))
	}
	if sentences2[0].Text != "Our refund policy allows you to return any item within 60 days if you are not satisfied." {
		t.Fatalf("unexpected sentence: %q", sentences2[0].Text)
	}
	if rem2 != "Next sentence begins" {
		t.Fatalf("unexpected remainder: %q", rem2)
	}
}

func TestExtractNumericalSpans(t *testing.T) {
	sentence := "We offer a 60-day return policy for $499/month at 18.5% interest."
	spans := ExtractNumericalSpans(sentence)

	if len(spans) != 3 {
		t.Fatalf("expected 3 spans, got %d: %+v", len(spans), spans)
	}

	// 1. 60-day duration (start 11)
	if spans[0].NumericValue != 60 || spans[0].ClaimType != "duration" {
		t.Fatalf("duration span mismatch: %+v", spans[0])
	}
	// 2. $499/month currency (start 36)
	if spans[1].NumericValue != 499 || spans[1].ClaimType != "currency" {
		t.Fatalf("currency span mismatch: %+v", spans[1])
	}
	// 3. 18.5% percentage (start 50)
	if spans[2].NumericValue != 18.5 || spans[2].ClaimType != "percentage" {
		t.Fatalf("percentage span mismatch: %+v", spans[2])
	}
}

func TestExtractClaims(t *testing.T) {
	store, err := registry.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seedPath := filepath.Join("..", "..", "examples", "ground_truth.json")
	if _, err := store.LoadFromFile(seedPath); err != nil {
		t.Fatal(err)
	}

	ext := NewExtractor(store)

	sentence := "Our refund return policy guarantees a full refund within 60 days."
	claims, err := ext.ExtractClaims(sentence, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d: %+v", len(claims), claims)
	}

	c := claims[0]
	if c.EntityID != "policy.refund" {
		t.Fatalf("expected entity policy.refund, got %s", c.EntityID)
	}
	if c.Attribute != "window_days" {
		t.Fatalf("expected attribute window_days, got %s", c.Attribute)
	}
	if c.NumericValue != 60 {
		t.Fatalf("expected numeric value 60, got %f", c.NumericValue)
	}
}
