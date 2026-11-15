package proxy

import (
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/registry"
	"github.com/alexandrmotologa/claimguard/internal/rewriter"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

func BenchmarkSentenceSplitting(b *testing.B) {
	text := "Thank you for reaching out to customer support. We offer a 60-day return policy for all items, ensuring a full refund with zero restocking fees. Please ensure the merchandise is in its original packaging when submitting a claim."
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = extractor.SplitSentences(text, false)
	}
}

func BenchmarkClaimExtraction(b *testing.B) {
	store, err := registry.NewStore(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	seedPath := filepath.Join("..", "..", "examples", "ground_truth.json")
	if _, err := store.LoadFromFile(seedPath); err != nil {
		b.Fatal(err)
	}

	ext := extractor.NewExtractor(store)
	sentence := "We offer a 60-day return policy for all items."
	prompt := "What is the return window?"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = ext.ExtractClaims(sentence, prompt)
	}
}

func BenchmarkFullVerificationPipeline(b *testing.B) {
	store, err := registry.NewStore(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()

	seedPath := filepath.Join("..", "..", "examples", "ground_truth.json")
	if _, err := store.LoadFromFile(seedPath); err != nil {
		b.Fatal(err)
	}

	cfg := config.DefaultConfig()
	ext := extractor.NewExtractor(store)
	ver := verifier.NewEngine(store, cfg.Policy)
	rew := rewriter.NewRewriter(cfg.Policy)

	sentence := "The enterprise plan costs $99/month with dedicated support."
	prompt := "How much does the Enterprise tier cost per month?"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		claims, _ := ext.ExtractClaims(sentence, prompt)
		var results []*verifier.VerificationResult
		for _, c := range claims {
			res, _ := ver.VerifyClaim(c)
			results = append(results, res)
		}
		_ = rew.Transform(sentence, results)
	}
}
