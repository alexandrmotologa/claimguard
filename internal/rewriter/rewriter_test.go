package rewriter

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/registry"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

func TestRewriteTransform(t *testing.T) {
	policy := config.PolicyConfig{
		DefaultAction:             "rewrite",
		DisclaimerTemplate:        " [Correction: Official enterprise policy is {{.ExpectedValue}}, not {{.AssertedValue}}]",
		StrictHaltMessage:         "Compliance violation: stream terminated.",
		NumericalTolerancePercent: 0.0,
	}

	rw := NewRewriter(policy)

	// 1. Test In-place Rewrite
	sentence := "We provide a 60-day return policy for all customers."
	// "60-day" is at index [13:19]
	start := strings.Index(sentence, "60-day")
	end := start + len("60-day")

	result := &verifier.VerificationResult{
		Status:        verifier.StatusContradicted,
		AppliedAction: "rewrite",
		Claim: extractor.ExtractedClaim{
			RawSentence:   sentence,
			AssertedValue: "60-day",
			Span: extractor.ValueSpan{
				RawMatch: "60-day",
				Start:    start,
				End:      end,
			},
		},
		AuthoritativeFact: &registry.Fact{
			ExpectedValue: "14 days",
		},
	}

	transformRes := rw.Transform(sentence, []*verifier.VerificationResult{result})
	if transformRes.ShouldHalt {
		t.Fatal("expected shouldHalt false")
	}
	expectedSentence := "We provide a 14-day return policy for all customers."
	if transformRes.TransformedSentence != expectedSentence {
		t.Fatalf("expected %q, got %q", expectedSentence, transformRes.TransformedSentence)
	}

	// 2. Test Disclaimer
	resultDisclaimer := &verifier.VerificationResult{
		Status:        verifier.StatusContradicted,
		AppliedAction: "disclaimer",
		Claim: extractor.ExtractedClaim{
			RawSentence:   sentence,
			EntityName:    "Refund Policy",
			Attribute:     "window_days",
			AssertedValue: "60 days",
		},
		AuthoritativeFact: &registry.Fact{
			ExpectedValue: "14 days",
		},
	}

	transformDisclaimer := rw.Transform(sentence, []*verifier.VerificationResult{resultDisclaimer})
	if !strings.Contains(transformDisclaimer.TransformedSentence, "[Correction: Official enterprise policy is 14 days, not 60 days]") {
		t.Fatalf("expected disclaimer in %q", transformDisclaimer.TransformedSentence)
	}

	// 3. Test Strict Halt
	resultHalt := &verifier.VerificationResult{
		Status:        verifier.StatusContradicted,
		AppliedAction: "strict_halt",
		AuditMessage:  "Critical deductible contradiction",
	}

	transformHalt := rw.Transform(sentence, []*verifier.VerificationResult{resultHalt})
	if !transformHalt.ShouldHalt {
		t.Fatal("expected shouldHalt true")
	}
	if transformHalt.TransformedSentence != policy.StrictHaltMessage {
		t.Fatalf("expected %q, got %q", policy.StrictHaltMessage, transformHalt.TransformedSentence)
	}
}
