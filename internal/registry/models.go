package registry

import "time"

// Entity represents a tracked enterprise subject, product, policy, or domain entity.
type Entity struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Category string   `json:"category"` // pricing, legal, fintech, insurance, sla, commerce
	Aliases  []string `json:"aliases"`  // Alternate names for token matching
}

// Fact represents an authoritative claim or numerical limit for an entity.
type Fact struct {
	ID            string    `json:"id"`
	EntityID      string    `json:"entity_id"`
	Attribute     string    `json:"attribute"`      // e.g. monthly_price, window_days, maximum_apr
	ClaimType     string    `json:"claim_type"`     // currency, duration, percentage, text
	ExpectedValue string    `json:"expected_value"` // Formatted canonical value, e.g. "$499/month", "14 days"
	NumericValue  float64   `json:"numeric_value"`  // Normalized float for arithmetic comparisons
	Unit          string    `json:"unit"`           // USD, days, %, etc.
	Tolerance     float64   `json:"tolerance"`      // Absolute or relative allowable deviation
	PolicyAction  string    `json:"policy_action"`  // rewrite, disclaimer, strict_halt, flag_only
	Description   string    `json:"description"`    // Factual context
	SourceDoc     string    `json:"source_doc"`     // Compliance source reference
	UpdatedAt     time.Time `json:"updated_at"`
}

// ForbiddenPattern represents a strictly barred assertion or unauthorized commitment.
type ForbiddenPattern struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Pattern      string    `json:"pattern"`       // Regex pattern (e.g. (?i)\blifetime\s+warranty\b)
	Category     string    `json:"category"`      // commercial, legal, medical, security
	PolicyAction string    `json:"policy_action"` // strict_halt, rewrite, disclaimer
	Replacement  string    `json:"replacement"`   // Replacement text if action is rewrite
	Reason       string    `json:"reason"`        // Compliance justification
	SourceDoc    string    `json:"source_doc"`    // Regulatory policy identifier
	UpdatedAt    time.Time `json:"updated_at"`
}

// ForbiddenMatch represents a detected occurrence of a forbidden pattern in a sentence.
type ForbiddenMatch struct {
	Pattern ForbiddenPattern `json:"pattern"`
	Matched string           `json:"matched"`
	Start   int              `json:"start"`
	End     int              `json:"end"`
}

// SeedData represents an importable knowledge registry bundle.
type SeedData struct {
	Entities          []Entity           `json:"entities" yaml:"entities"`
	Facts             []Fact             `json:"facts" yaml:"facts"`
	ForbiddenPatterns []ForbiddenPattern `json:"forbidden_patterns,omitempty" yaml:"forbidden_patterns,omitempty"`
}
