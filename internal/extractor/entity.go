package extractor

import (
	"strings"

	"github.com/alexandrmotologa/claimguard/internal/registry"
)

// ExtractedClaim represents a single factual or numerical assertion ready for verification.
type ExtractedClaim struct {
	RawSentence   string     `json:"raw_sentence"`
	EntityID      string     `json:"entity_id"`
	EntityName    string     `json:"entity_name"`
	Attribute     string     `json:"attribute"`
	ClaimType     string     `json:"claim_type"`
	AssertedValue string     `json:"asserted_value"`
	NumericValue  float64    `json:"numeric_value"`
	Unit          string     `json:"unit"`
	Span          ValueSpan  `json:"span"`
}

// Extractor coordinates sentence parsing, numerical extraction, and entity association.
type Extractor struct {
	store *registry.Store
}

// NewExtractor creates a new semantic claim extractor.
func NewExtractor(store *registry.Store) *Extractor {
	return &Extractor{store: store}
}

// ExtractClaims scans a sentence, finds numerical spans, and associates them with registered enterprise facts.
// promptContext can optionally provide prior conversation context to disambiguate entities.
func (e *Extractor) ExtractClaims(sentence string, promptContext string) ([]ExtractedClaim, error) {
	var claims []ExtractedClaim

	spans := ExtractNumericalSpans(sentence)
	if len(spans) == 0 {
		return claims, nil
	}

	combinedContext := promptContext + " " + sentence

	// Attempt to match entity from sentence first, falling back to promptContext
	matchedEntity, err := e.store.MatchEntity(sentence)
	if err != nil {
		return nil, err
	}
	if matchedEntity == nil && promptContext != "" {
		matchedEntity, err = e.store.MatchEntity(promptContext)
		if err != nil {
			return nil, err
		}
	}

	if matchedEntity == nil {
		return claims, nil
	}

	// Retrieve facts for the matched entity
	facts, err := e.store.FindFactsByEntity(matchedEntity.ID)
	if err != nil {
		return nil, err
	}
	if len(facts) == 0 {
		return claims, nil
	}

	lowerContext := strings.ToLower(combinedContext)

	for _, span := range spans {
		var candidateFact *registry.Fact

		// Try matching by claim type first
		for i := range facts {
			f := &facts[i]
			if f.ClaimType == span.ClaimType {
				candidateFact = f
				break
			}
		}

		// Disambiguate if multiple facts or attribute keywords present in context
		for i := range facts {
			f := &facts[i]
			if f.ClaimType == span.ClaimType && strings.Contains(lowerContext, strings.ToLower(f.Attribute)) {
				candidateFact = f
				break
			}
		}

		if candidateFact == nil {
			// If claim types didn't match strictly, check if entity only has 1 fact
			if len(facts) == 1 {
				candidateFact = &facts[0]
			} else {
				continue
			}
		}

		claims = append(claims, ExtractedClaim{
			RawSentence:   sentence,
			EntityID:      matchedEntity.ID,
			EntityName:    matchedEntity.Name,
			Attribute:     candidateFact.Attribute,
			ClaimType:     candidateFact.ClaimType,
			AssertedValue: span.RawMatch,
			NumericValue:  span.NumericValue,
			Unit:          span.Unit,
			Span:          span,
		})
	}

	return claims, nil
}
