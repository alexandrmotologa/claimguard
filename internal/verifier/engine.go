package verifier

import (
	"fmt"
	"math"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/registry"
)

// VerificationStatus represents the result of validating a claim.
type VerificationStatus string

const (
	StatusVerified     VerificationStatus = "VERIFIED"
	StatusContradicted VerificationStatus = "CONTRADICTED"
	StatusUngrounded   VerificationStatus = "UNGROUNDED"
)

// VerificationResult encapsulates the audit outcome of a single claim verification.
type VerificationResult struct {
	Status             VerificationStatus        `json:"status"`
	Claim              extractor.ExtractedClaim  `json:"claim"`
	AuthoritativeFact  *registry.Fact            `json:"authoritative_fact,omitempty"`
	Delta              float64                   `json:"delta"`
	AppliedAction      string                    `json:"applied_action"` // rewrite, disclaimer, strict_halt, flag_only
	AuditMessage       string                    `json:"audit_message"`
	CorrectedSentence  string                    `json:"corrected_sentence,omitempty"`
}

// Engine performs factual verification of claims against the Ground Truth store.
type Engine struct {
	store  *registry.Store
	policy config.PolicyConfig
}

// NewEngine creates a new verification engine instance.
func NewEngine(store *registry.Store, policy config.PolicyConfig) *Engine {
	return &Engine{
		store:  store,
		policy: policy,
	}
}

// UpdatePolicy dynamically updates the runtime action policy.
func (e *Engine) UpdatePolicy(p config.PolicyConfig) {
	e.policy = p
}

// GetPolicy returns the current active policy configuration.
func (e *Engine) GetPolicy() config.PolicyConfig {
	return e.policy
}

// VerifyClaim checks an extracted claim against authoritative enterprise facts.
func (e *Engine) VerifyClaim(claim extractor.ExtractedClaim) (*VerificationResult, error) {
	fact, err := e.store.FindFactByEntityAndAttribute(claim.EntityID, claim.Attribute)
	if err != nil {
		return nil, fmt.Errorf("lookup fact: %w", err)
	}

	if fact == nil {
		return &VerificationResult{
			Status:        StatusUngrounded,
			Claim:         claim,
			AppliedAction: "flag_only",
			AuditMessage:  fmt.Sprintf("No authoritative ground truth fact recorded for %s.%s", claim.EntityID, claim.Attribute),
		}, nil
	}

	delta := math.Abs(claim.NumericValue - fact.NumericValue)
	isMatch := IsWithinTolerance(claim.NumericValue, fact.NumericValue, fact.Tolerance, e.policy.NumericalTolerancePercent)

	if isMatch {
		return &VerificationResult{
			Status:            StatusVerified,
			Claim:             claim,
			AuthoritativeFact: fact,
			Delta:             delta,
			AppliedAction:     "none",
			AuditMessage:      fmt.Sprintf("Claim verified against %s (expected %s, asserted %s)", fact.ID, fact.ExpectedValue, claim.AssertedValue),
		}, nil
	}

	// Contradiction detected. Determine action policy.
	action := e.policy.DefaultAction
	if fact.PolicyAction != "" {
		action = fact.PolicyAction
	}

	return &VerificationResult{
		Status:            StatusContradicted,
		Claim:             claim,
		AuthoritativeFact: fact,
		Delta:             delta,
		AppliedAction:     action,
		AuditMessage: fmt.Sprintf("Factual contradiction: asserted %s != authoritative %s for %s.%s",
			claim.AssertedValue, fact.ExpectedValue, claim.EntityID, claim.Attribute),
	}, nil
}
