package verifier

import (
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/registry"
)

func TestVerifyClaim(t *testing.T) {
	store, err := registry.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	seedPath := filepath.Join("..", "..", "examples", "ground_truth.json")
	if _, err := store.LoadFromFile(seedPath); err != nil {
		t.Fatal(err)
	}

	policy := config.PolicyConfig{
		DefaultAction:             "rewrite",
		DisclaimerTemplate:        " [Correction: Official enterprise policy is {{.ExpectedValue}}, not {{.AssertedValue}}]",
		StrictHaltMessage:         "Compliance violation",
		NumericalTolerancePercent: 0.0,
	}

	engine := NewEngine(store, policy)

	// Case 1: Contradicted claim (asserting 60 days when policy is 14 days)
	claimContradicted := extractor.ExtractedClaim{
		RawSentence:   "Our return policy is 60 days.",
		EntityID:      "policy.refund",
		EntityName:    "Refund Policy",
		Attribute:     "window_days",
		ClaimType:     "duration",
		AssertedValue: "60 days",
		NumericValue:  60,
		Unit:          "days",
		Span: extractor.ValueSpan{
			RawMatch:     "60 days",
			NumericValue: 60,
			Unit:         "days",
			ClaimType:    "duration",
			Start:        21,
			End:          28,
		},
	}

	res1, err := engine.VerifyClaim(claimContradicted)
	if err != nil {
		t.Fatal(err)
	}
	if res1.Status != StatusContradicted {
		t.Fatalf("expected CONTRADICTED, got %s", res1.Status)
	}
	if res1.AppliedAction != "rewrite" {
		t.Fatalf("expected rewrite, got %s", res1.AppliedAction)
	}

	// Case 2: Verified claim (asserting 14 days)
	claimVerified := claimContradicted
	claimVerified.AssertedValue = "14 days"
	claimVerified.NumericValue = 14

	res2, err := engine.VerifyClaim(claimVerified)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Status != StatusVerified {
		t.Fatalf("expected VERIFIED, got %s", res2.Status)
	}

	// Case 3: Strict halt claim (insurance deductible contradiction)
	claimHalt := extractor.ExtractedClaim{
		RawSentence:   "Your medical deductible is $250.",
		EntityID:      "insurance.deductible",
		Attribute:     "individual_amount",
		ClaimType:     "currency",
		AssertedValue: "$250",
		NumericValue:  250,
		Unit:          "USD",
	}

	res3, err := engine.VerifyClaim(claimHalt)
	if err != nil {
		t.Fatal(err)
	}
	if res3.Status != StatusContradicted {
		t.Fatalf("expected CONTRADICTED, got %s", res3.Status)
	}
	if res3.AppliedAction != "strict_halt" {
		t.Fatalf("expected strict_halt, got %s", res3.AppliedAction)
	}
}
