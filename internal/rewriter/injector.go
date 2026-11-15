package rewriter

import (
	"sort"
	"strings"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

// Rewriter coordinates in-flight stream modifications, disclaimer injections, and strict refusals.
type Rewriter struct {
	policy config.PolicyConfig
}

// NewRewriter creates a new active stream rewriter.
func NewRewriter(policy config.PolicyConfig) *Rewriter {
	return &Rewriter{policy: policy}
}

// UpdatePolicy updates the active rewriter policy.
func (r *Rewriter) UpdatePolicy(p config.PolicyConfig) {
	r.policy = p
}

// TransformationResult holds the rewritten sentence, the primary action taken, and halt status.
type TransformationResult struct {
	TransformedSentence string
	PrimaryAction       string // verified, rewrite, disclaimer, strict_halt, flag_only
	ShouldHalt          bool
	HaltReason          string
}

// Transform processes a sentence against its verification results.
func (r *Rewriter) Transform(sentence string, results []*verifier.VerificationResult) TransformationResult {
	if len(results) == 0 {
		return TransformationResult{
			TransformedSentence: sentence,
			PrimaryAction:       "verified",
			ShouldHalt:          false,
		}
	}

	// 1. Check for strict_halt first
	for _, res := range results {
		if res.Status == verifier.StatusContradicted && res.AppliedAction == "strict_halt" {
			return TransformationResult{
				TransformedSentence: r.policy.StrictHaltMessage,
				PrimaryAction:       "strict_halt",
				ShouldHalt:          true,
				HaltReason:          res.AuditMessage,
			}
		}
	}

	// 2. Separate contradictions by action
	var rewrites []*verifier.VerificationResult
	var disclaimers []*verifier.VerificationResult
	hasContradiction := false

	for _, res := range results {
		if res.Status == verifier.StatusContradicted {
			hasContradiction = true
			switch res.AppliedAction {
			case "rewrite":
				rewrites = append(rewrites, res)
			case "disclaimer":
				disclaimers = append(disclaimers, res)
			}
		}
	}

	if !hasContradiction {
		return TransformationResult{
			TransformedSentence: sentence,
			PrimaryAction:       "verified",
			ShouldHalt:          false,
		}
	}

	currentSentence := sentence

	// 3. Apply in-place rewrites in descending order of span.Start to prevent offset shifting
	if len(rewrites) > 0 {
		sort.Slice(rewrites, func(i, j int) bool {
			return rewrites[i].Claim.Span.Start > rewrites[j].Claim.Span.Start
		})

		for _, res := range rewrites {
			if res.AuthoritativeFact == nil {
				continue
			}

			span := res.Claim.Span
			if span.Start < 0 || span.End > len(currentSentence) || span.Start >= span.End {
				continue
			}

			replacement := res.AuthoritativeFact.ExpectedValue

			// Handle hyphenated duration adjective (e.g. "60-day" -> "14-day")
			if strings.Contains(span.RawMatch, "-") && strings.HasSuffix(span.RawMatch, "day") {
				replacement = strings.ReplaceAll(replacement, " days", "-day")
			}

			currentSentence = currentSentence[:span.Start] + replacement + currentSentence[span.End:]
			res.CorrectedSentence = currentSentence
		}
	}

	// 4. Append disclaimers if any
	primaryAction := "rewrite"
	if len(disclaimers) > 0 {
		primaryAction = "disclaimer"
		for _, res := range disclaimers {
			if res.AuthoritativeFact == nil {
				continue
			}

			data := DisclaimerData{
				EntityName:    res.Claim.EntityName,
				Attribute:     res.Claim.Attribute,
				AssertedValue: res.Claim.AssertedValue,
				ExpectedValue: res.AuthoritativeFact.ExpectedValue,
				SourceDoc:     res.AuthoritativeFact.SourceDoc,
			}

			rendered, err := RenderDisclaimer(r.policy.DisclaimerTemplate, data)
			if err == nil {
				// Strip trailing punctuation from sentence before appending or append cleanly
				currentSentence = strings.TrimRight(currentSentence, " \t\r\n") + rendered
			}
			res.CorrectedSentence = currentSentence
		}
	}

	return TransformationResult{
		TransformedSentence: currentSentence,
		PrimaryAction:       primaryAction,
		ShouldHalt:          false,
	}
}
