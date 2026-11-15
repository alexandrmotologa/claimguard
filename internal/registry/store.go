package registry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages persistence and querying of ground truth enterprise facts.
type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewStore initializes a SQLite-backed registry.
// Pass ":memory:" for high-speed in-memory operations or a file path for persistence.
func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		dbPath = ":memory:"
	}

	dsn := dbPath
	if dbPath == ":memory:" {
		// Use shared in-memory URI so multiple connections can access the same memory DB if needed
		dsn = "file:claimguard_mem?mode=memory&cache=shared"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// For in-memory shared database, maintain at least 1 open idle connection
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return s, nil
}

// initSchema creates the entities and facts tables if they do not exist.
func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS entities (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		aliases_json TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_entities_category ON entities(category);

	CREATE TABLE IF NOT EXISTS facts (
		id TEXT PRIMARY KEY,
		entity_id TEXT NOT NULL,
		attribute TEXT NOT NULL,
		claim_type TEXT NOT NULL,
		expected_value TEXT NOT NULL,
		numeric_value REAL NOT NULL,
		unit TEXT NOT NULL,
		tolerance REAL NOT NULL DEFAULT 0.0,
		policy_action TEXT NOT NULL DEFAULT 'rewrite',
		description TEXT NOT NULL DEFAULT '',
		source_doc TEXT NOT NULL DEFAULT '',
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (entity_id) REFERENCES entities(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_facts_entity ON facts(entity_id);
	CREATE INDEX IF NOT EXISTS idx_facts_attribute ON facts(attribute);

	CREATE TABLE IF NOT EXISTS forbidden_patterns (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		pattern TEXT NOT NULL,
		category TEXT NOT NULL,
		policy_action TEXT NOT NULL DEFAULT 'strict_halt',
		replacement TEXT NOT NULL DEFAULT '',
		reason TEXT NOT NULL DEFAULT '',
		source_doc TEXT NOT NULL DEFAULT '',
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_forbidden_category ON forbidden_patterns(category);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveEntity inserts or updates an entity definition.
func (s *Store) SaveEntity(e Entity) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	aliasesJSON, err := json.Marshal(e.Aliases)
	if err != nil {
		aliasesJSON = []byte("[]")
	}

	query := `
	INSERT INTO entities (id, name, category, aliases_json)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		category = excluded.category,
		aliases_json = excluded.aliases_json;
	`
	_, err = s.db.Exec(query, e.ID, e.Name, e.Category, string(aliasesJSON))
	return err
}

// GetEntity retrieves an entity by its identifier.
func (s *Store) GetEntity(id string) (*Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var e Entity
	var aliasesJSON string

	query := `SELECT id, name, category, aliases_json FROM entities WHERE id = ?`
	err := s.db.QueryRow(query, id).Scan(&e.ID, &e.Name, &e.Category, &aliasesJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(aliasesJSON), &e.Aliases); err != nil {
		e.Aliases = []string{}
	}

	return &e, nil
}

// ListEntities returns all registered entities.
func (s *Store) ListEntities() ([]Entity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT id, name, category, aliases_json FROM entities ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Entity
	for rows.Next() {
		var e Entity
		var aliasesJSON string
		if err := rows.Scan(&e.ID, &e.Name, &e.Category, &aliasesJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(aliasesJSON), &e.Aliases); err != nil {
			e.Aliases = []string{}
		}
		list = append(list, e)
	}

	return list, rows.Err()
}

// DeleteEntity removes an entity and its associated facts.
func (s *Store) DeleteEntity(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM entities WHERE id = ?`, id)
	return err
}

// SaveFact inserts or updates an authoritative fact.
func (s *Store) SaveFact(f Fact) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if f.UpdatedAt.IsZero() {
		f.UpdatedAt = time.Now().UTC()
	}

	query := `
	INSERT INTO facts (
		id, entity_id, attribute, claim_type, expected_value,
		numeric_value, unit, tolerance, policy_action, description,
		source_doc, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		entity_id = excluded.entity_id,
		attribute = excluded.attribute,
		claim_type = excluded.claim_type,
		expected_value = excluded.expected_value,
		numeric_value = excluded.numeric_value,
		unit = excluded.unit,
		tolerance = excluded.tolerance,
		policy_action = excluded.policy_action,
		description = excluded.description,
		source_doc = excluded.source_doc,
		updated_at = excluded.updated_at;
	`

	_, err := s.db.Exec(query,
		f.ID, f.EntityID, f.Attribute, f.ClaimType, f.ExpectedValue,
		f.NumericValue, f.Unit, f.Tolerance, f.PolicyAction, f.Description,
		f.SourceDoc, f.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetFact retrieves a single fact by ID.
func (s *Store) GetFact(id string) (*Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var f Fact
	var updatedAtStr string

	query := `
	SELECT id, entity_id, attribute, claim_type, expected_value,
	       numeric_value, unit, tolerance, policy_action, description,
	       source_doc, updated_at
	FROM facts WHERE id = ?
	`

	err := s.db.QueryRow(query, id).Scan(
		&f.ID, &f.EntityID, &f.Attribute, &f.ClaimType, &f.ExpectedValue,
		&f.NumericValue, &f.Unit, &f.Tolerance, &f.PolicyAction, &f.Description,
		&f.SourceDoc, &updatedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	t, _ := time.Parse(time.RFC3339, updatedAtStr)
	f.UpdatedAt = t
	return &f, nil
}

// ListFacts returns all registered authoritative facts.
func (s *Store) ListFacts() ([]Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT id, entity_id, attribute, claim_type, expected_value,
	       numeric_value, unit, tolerance, policy_action, description,
	       source_doc, updated_at
	FROM facts ORDER BY entity_id, attribute ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Fact
	for rows.Next() {
		var f Fact
		var updatedAtStr string
		err := rows.Scan(
			&f.ID, &f.EntityID, &f.Attribute, &f.ClaimType, &f.ExpectedValue,
			&f.NumericValue, &f.Unit, &f.Tolerance, &f.PolicyAction, &f.Description,
			&f.SourceDoc, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339, updatedAtStr)
		f.UpdatedAt = t
		list = append(list, f)
	}

	return list, rows.Err()
}

// FindFactsByEntity retrieves all facts associated with a specific entity.
func (s *Store) FindFactsByEntity(entityID string) ([]Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT id, entity_id, attribute, claim_type, expected_value,
	       numeric_value, unit, tolerance, policy_action, description,
	       source_doc, updated_at
	FROM facts WHERE entity_id = ?
	`

	rows, err := s.db.Query(query, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Fact
	for rows.Next() {
		var f Fact
		var updatedAtStr string
		err := rows.Scan(
			&f.ID, &f.EntityID, &f.Attribute, &f.ClaimType, &f.ExpectedValue,
			&f.NumericValue, &f.Unit, &f.Tolerance, &f.PolicyAction, &f.Description,
			&f.SourceDoc, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339, updatedAtStr)
		f.UpdatedAt = t
		list = append(list, f)
	}

	return list, rows.Err()
}

// FindFactByEntityAndAttribute finds a specific fact for an entity and attribute.
func (s *Store) FindFactByEntityAndAttribute(entityID, attribute string) (*Fact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var f Fact
	var updatedAtStr string

	query := `
	SELECT id, entity_id, attribute, claim_type, expected_value,
	       numeric_value, unit, tolerance, policy_action, description,
	       source_doc, updated_at
	FROM facts WHERE entity_id = ? AND attribute = ?
	LIMIT 1
	`

	err := s.db.QueryRow(query, entityID, attribute).Scan(
		&f.ID, &f.EntityID, &f.Attribute, &f.ClaimType, &f.ExpectedValue,
		&f.NumericValue, &f.Unit, &f.Tolerance, &f.PolicyAction, &f.Description,
		&f.SourceDoc, &updatedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	t, _ := time.Parse(time.RFC3339, updatedAtStr)
	f.UpdatedAt = t
	return &f, nil
}

// DeleteFact removes a single fact by ID.
func (s *Store) DeleteFact(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM facts WHERE id = ?`, id)
	return err
}

// MatchEntity scans a sentence or phrase against known entity names and aliases.
// Returns the best matching entity, or nil if no match is found.
func (s *Store) MatchEntity(text string) (*Entity, error) {
	lowerText := strings.ToLower(text)
	entities, err := s.ListEntities()
	if err != nil {
		return nil, err
	}

	var bestEntity *Entity
	longestMatch := 0

	for i := range entities {
		e := &entities[i]
		// Check direct name match
		if strings.Contains(lowerText, strings.ToLower(e.Name)) {
			if len(e.Name) > longestMatch {
				longestMatch = len(e.Name)
				bestEntity = e
			}
		}
		// Check aliases
		for _, alias := range e.Aliases {
			if strings.Contains(lowerText, strings.ToLower(alias)) {
				if len(alias) > longestMatch {
					longestMatch = len(alias)
					bestEntity = e
				}
			}
		}
	}

	return bestEntity, nil
}

// SaveForbiddenPattern inserts or updates a prohibited assertion rule.
func (s *Store) SaveForbiddenPattern(fp ForbiddenPattern) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fp.UpdatedAt.IsZero() {
		fp.UpdatedAt = time.Now().UTC()
	}

	query := `
	INSERT INTO forbidden_patterns (
		id, name, pattern, category, policy_action, replacement, reason, source_doc, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		pattern = excluded.pattern,
		category = excluded.category,
		policy_action = excluded.policy_action,
		replacement = excluded.replacement,
		reason = excluded.reason,
		source_doc = excluded.source_doc,
		updated_at = excluded.updated_at;
	`
	_, err := s.db.Exec(query,
		fp.ID, fp.Name, fp.Pattern, fp.Category, fp.PolicyAction,
		fp.Replacement, fp.Reason, fp.SourceDoc, fp.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetForbiddenPattern retrieves a forbidden pattern rule by its ID.
func (s *Store) GetForbiddenPattern(id string) (*ForbiddenPattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var fp ForbiddenPattern
	var updatedAtStr string

	query := `
	SELECT id, name, pattern, category, policy_action, replacement, reason, source_doc, updated_at
	FROM forbidden_patterns WHERE id = ?
	`
	err := s.db.QueryRow(query, id).Scan(
		&fp.ID, &fp.Name, &fp.Pattern, &fp.Category, &fp.PolicyAction,
		&fp.Replacement, &fp.Reason, &fp.SourceDoc, &updatedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t, _ := time.Parse(time.RFC3339, updatedAtStr)
	fp.UpdatedAt = t
	return &fp, nil
}

// ListForbiddenPatterns returns all active forbidden commitment rules.
func (s *Store) ListForbiddenPatterns() ([]ForbiddenPattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
	SELECT id, name, pattern, category, policy_action, replacement, reason, source_doc, updated_at
	FROM forbidden_patterns ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ForbiddenPattern
	for rows.Next() {
		var fp ForbiddenPattern
		var updatedAtStr string
		err := rows.Scan(
			&fp.ID, &fp.Name, &fp.Pattern, &fp.Category, &fp.PolicyAction,
			&fp.Replacement, &fp.Reason, &fp.SourceDoc, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339, updatedAtStr)
		fp.UpdatedAt = t
		list = append(list, fp)
	}

	return list, rows.Err()
}

// DeleteForbiddenPattern removes a forbidden rule by ID.
func (s *Store) DeleteForbiddenPattern(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM forbidden_patterns WHERE id = ?`, id)
	return err
}

// CheckForbidden scans a sentence against all registered forbidden patterns.
func (s *Store) CheckForbidden(sentence string) ([]ForbiddenMatch, error) {
	patterns, err := s.ListForbiddenPatterns()
	if err != nil {
		return nil, err
	}

	var matches []ForbiddenMatch
	for _, fp := range patterns {
		re, err := regexp.Compile(fp.Pattern)
		if err != nil {
			continue
		}

		locs := re.FindAllStringIndex(sentence, -1)
		for _, loc := range locs {
			matches = append(matches, ForbiddenMatch{
				Pattern: fp,
				Matched: sentence[loc[0]:loc[1]],
				Start:   loc[0],
				End:     loc[1],
			})
		}
	}

	return matches, nil
}

// Close terminates open database handles.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}
