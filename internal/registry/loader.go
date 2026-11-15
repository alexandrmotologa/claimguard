package registry

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadFromFile ingests entities and facts from a file into the store.
// Supports .json, .yaml, .yml, and .csv formats.
func (s *Store) LoadFromFile(path string) (*SeedData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read seed file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var seed SeedData

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &seed); err != nil {
			return nil, fmt.Errorf("parse json seed: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &seed); err != nil {
			return nil, fmt.Errorf("parse yaml seed: %w", err)
		}
	case ".csv":
		parsedSeed, err := parseCSV(strings.NewReader(string(data)))
		if err != nil {
			return nil, fmt.Errorf("parse csv seed: %w", err)
		}
		seed = *parsedSeed
	default:
		return nil, fmt.Errorf("unsupported seed file format %s", ext)
	}

	// Persist entities
	for _, e := range seed.Entities {
		if err := s.SaveEntity(e); err != nil {
			return nil, fmt.Errorf("save entity %s: %w", e.ID, err)
		}
	}

	// Persist facts
	for _, f := range seed.Facts {
		if err := s.SaveFact(f); err != nil {
			return nil, fmt.Errorf("save fact %s: %w", f.ID, err)
		}
	}

	// Persist forbidden commitment patterns
	for _, fp := range seed.ForbiddenPatterns {
		if err := s.SaveForbiddenPattern(fp); err != nil {
			return nil, fmt.Errorf("save forbidden pattern %s: %w", fp.ID, err)
		}
	}

	return &seed, nil
}

// parseCSV converts tabular CSV rows into entities and facts.
// Expected CSV format:
// entity_id,entity_name,category,aliases,fact_id,attribute,claim_type,expected_value,numeric_value,unit,tolerance,policy_action,description,source_doc
func parseCSV(r io.Reader) (*SeedData, error) {
	reader := csv.NewReader(r)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return &SeedData{}, nil
	}

	entityMap := make(map[string]Entity)
	var facts []Fact

	// Skip header row
	for _, row := range rows[1:] {
		if len(row) < 14 {
			continue
		}
		entityID := strings.TrimSpace(row[0])
		entityName := strings.TrimSpace(row[1])
		category := strings.TrimSpace(row[2])
		aliasesRaw := strings.TrimSpace(row[3])
		factID := strings.TrimSpace(row[4])
		attribute := strings.TrimSpace(row[5])
		claimType := strings.TrimSpace(row[6])
		expectedVal := strings.TrimSpace(row[7])
		numVal, _ := strconv.ParseFloat(strings.TrimSpace(row[8]), 64)
		unit := strings.TrimSpace(row[9])
		tolerance, _ := strconv.ParseFloat(strings.TrimSpace(row[10]), 64)
		policyAction := strings.TrimSpace(row[11])
		desc := strings.TrimSpace(row[12])
		sourceDoc := strings.TrimSpace(row[13])

		var aliases []string
		if aliasesRaw != "" {
			for _, a := range strings.Split(aliasesRaw, ";") {
				if trimmed := strings.TrimSpace(a); trimmed != "" {
					aliases = append(aliases, trimmed)
				}
			}
		}

		if _, exists := entityMap[entityID]; !exists {
			entityMap[entityID] = Entity{
				ID:       entityID,
				Name:     entityName,
				Category: category,
				Aliases:  aliases,
			}
		}

		facts = append(facts, Fact{
			ID:            factID,
			EntityID:      entityID,
			Attribute:     attribute,
			ClaimType:     claimType,
			ExpectedValue: expectedVal,
			NumericValue:  numVal,
			Unit:          unit,
			Tolerance:     tolerance,
			PolicyAction:  policyAction,
			Description:   desc,
			SourceDoc:     sourceDoc,
			UpdatedAt:     time.Now().UTC(),
		})
	}

	var entities []Entity
	for _, e := range entityMap {
		entities = append(entities, e)
	}

	return &SeedData{Entities: entities, Facts: facts}, nil
}
