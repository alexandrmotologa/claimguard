package registry

import (
	"path/filepath"
	"testing"
)

func TestStoreBasics(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	e := Entity{
		ID:       "test.policy",
		Name:     "Test Policy",
		Category: "legal",
		Aliases:  []string{"policy test", "guarantee"},
	}
	if err := s.SaveEntity(e); err != nil {
		t.Fatalf("save entity: %v", err)
	}

	gotE, err := s.GetEntity("test.policy")
	if err != nil {
		t.Fatalf("get entity: %v", err)
	}
	if gotE == nil || gotE.Name != "Test Policy" {
		t.Fatalf("unexpected entity: %+v", gotE)
	}

	f := Fact{
		ID:            "fact-1",
		EntityID:      "test.policy",
		Attribute:     "window_days",
		ClaimType:     "duration",
		ExpectedValue: "14 days",
		NumericValue:  14,
		Unit:          "days",
		Tolerance:     0,
		PolicyAction:  "rewrite",
	}
	if err := s.SaveFact(f); err != nil {
		t.Fatalf("save fact: %v", err)
	}

	gotF, err := s.GetFact("fact-1")
	if err != nil {
		t.Fatalf("get fact: %v", err)
	}
	if gotF == nil || gotF.NumericValue != 14 {
		t.Fatalf("unexpected fact: %+v", gotF)
	}

	matched, err := s.MatchEntity("Do you offer a money guarantee for your customers?")
	if err != nil {
		t.Fatalf("match entity: %v", err)
	}
	if matched == nil || matched.ID != "test.policy" {
		t.Fatalf("expected test.policy, got %+v", matched)
	}
}

func TestLoadSeedFile(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	seedPath := filepath.Join("..", "..", "examples", "ground_truth.json")
	seed, err := s.LoadFromFile(seedPath)
	if err != nil {
		t.Fatalf("load seed file: %v", err)
	}

	if len(seed.Entities) == 0 || len(seed.Facts) == 0 {
		t.Fatalf("expected non-empty seed, got %d entities, %d facts", len(seed.Entities), len(seed.Facts))
	}

	// Verify refund policy entity
	matched, err := s.MatchEntity("What is your refund policy window?")
	if err != nil {
		t.Fatalf("match entity: %v", err)
	}
	if matched == nil || matched.ID != "policy.refund" {
		t.Fatalf("expected policy.refund, got %+v", matched)
	}

	fact, err := s.FindFactByEntityAndAttribute("policy.refund", "window_days")
	if err != nil {
		t.Fatalf("find fact: %v", err)
	}
	if fact == nil || fact.NumericValue != 14 {
		t.Fatalf("expected fact with 14 days, got %+v", fact)
	}
}
