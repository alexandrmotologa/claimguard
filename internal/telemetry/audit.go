package telemetry

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// AuditEvent represents a tamper-evident compliance audit record.
type AuditEvent struct {
	ID                 string    `json:"id"`
	Timestamp          time.Time `json:"timestamp"`
	RequestID          string    `json:"request_id"`
	PromptPreview      string    `json:"prompt_preview"`
	OriginalSentence   string    `json:"original_sentence"`
	CorrectedSentence  string    `json:"corrected_sentence"`
	EntityID           string    `json:"entity_id"`
	Attribute          string    `json:"attribute"`
	AssertedValue      string    `json:"asserted_value"`
	AuthoritativeValue string    `json:"authoritative_value"`
	PolicyAction       string    `json:"policy_action"`
	Status             string    `json:"status"` // VERIFIED, CONTRADICTED, UNGROUNDED
	LatencyMs          float64   `json:"latency_ms"`
	PayloadHash        string    `json:"payload_hash"`
}

// ComputePayloadHash calculates a deterministic SHA-256 digest of the audit event.
func (e *AuditEvent) ComputePayloadHash() string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%.2f",
		e.RequestID,
		e.Timestamp.UTC().Format(time.RFC3339),
		e.PromptPreview,
		e.OriginalSentence,
		e.CorrectedSentence,
		e.EntityID,
		e.Attribute,
		e.AssertedValue,
		e.AuthoritativeValue,
		e.PolicyAction,
		e.Status,
		e.LatencyMs,
	)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// AuditLogger manages persistent storage and cryptographic verification of compliance events.
type AuditLogger struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewAuditLogger initializes the SQLite audit logger.
func NewAuditLogger(dbPath string) (*AuditLogger, error) {
	if dbPath == "" {
		dbPath = "claimguard_audit.db"
	}

	dsn := dbPath
	if dbPath == ":memory:" {
		dsn = "file:claimguard_audit_mem?mode=memory&cache=shared"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open audit sqlite: %w", err)
	}

	logger := &AuditLogger{db: db}
	if err := logger.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("init audit schema: %w", err)
	}

	return logger, nil
}

func (l *AuditLogger) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_events (
		id TEXT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		request_id TEXT NOT NULL,
		prompt_preview TEXT NOT NULL,
		original_sentence TEXT NOT NULL,
		corrected_sentence TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		attribute TEXT NOT NULL,
		asserted_value TEXT NOT NULL,
		authoritative_value TEXT NOT NULL,
		policy_action TEXT NOT NULL,
		status TEXT NOT NULL,
		latency_ms REAL NOT NULL,
		payload_hash TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_status ON audit_events(status);
	CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_events(entity_id);
	`
	_, err := l.db.Exec(schema)
	return err
}

// RecordEvent stores an audit event with an automatically calculated payload hash.
func (l *AuditLogger) RecordEvent(event AuditEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.PayloadHash == "" {
		event.PayloadHash = event.ComputePayloadHash()
	}

	query := `
	INSERT INTO audit_events (
		id, timestamp, request_id, prompt_preview, original_sentence,
		corrected_sentence, entity_id, attribute, asserted_value,
		authoritative_value, policy_action, status, latency_ms, payload_hash
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := l.db.Exec(query,
		event.ID,
		event.Timestamp.Format(time.RFC3339),
		event.RequestID,
		event.PromptPreview,
		event.OriginalSentence,
		event.CorrectedSentence,
		event.EntityID,
		event.Attribute,
		event.AssertedValue,
		event.AuthoritativeValue,
		event.PolicyAction,
		event.Status,
		event.LatencyMs,
		event.PayloadHash,
	)
	return err
}

// ListEvents returns a paginated slice of audit events with optional filters.
func (l *AuditLogger) ListEvents(limit, offset int, statusFilter, entityFilter string) ([]AuditEvent, int, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	baseQuery := `FROM audit_events WHERE 1=1`
	var args []interface{}

	if statusFilter != "" {
		baseQuery += ` AND status = ?`
		args = append(args, statusFilter)
	}
	if entityFilter != "" {
		baseQuery += ` AND entity_id = ?`
		args = append(args, entityFilter)
	}

	// Count total
	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int
	if err := l.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Select rows
	selectQuery := `
	SELECT id, timestamp, request_id, prompt_preview, original_sentence,
	       corrected_sentence, entity_id, attribute, asserted_value,
	       authoritative_value, policy_action, status, latency_ms, payload_hash
	` + baseQuery + ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`

	selectArgs := append(args, limit, offset)
	rows, err := l.db.Query(selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		var tsStr string
		err := rows.Scan(
			&e.ID, &tsStr, &e.RequestID, &e.PromptPreview, &e.OriginalSentence,
			&e.CorrectedSentence, &e.EntityID, &e.Attribute, &e.AssertedValue,
			&e.AuthoritativeValue, &e.PolicyAction, &e.Status, &e.LatencyMs, &e.PayloadHash,
		)
		if err != nil {
			return nil, 0, err
		}
		t, _ := time.Parse(time.RFC3339, tsStr)
		e.Timestamp = t
		events = append(events, e)
	}

	return events, total, rows.Err()
}

// ExportEvents exports audit events as JSON or CSV bytes.
func (l *AuditLogger) ExportEvents(format string) ([]byte, error) {
	events, _, err := l.ListEvents(10000, 0, "", "")
	if err != nil {
		return nil, err
	}

	if format == "csv" {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		w.Write([]string{
			"id", "timestamp", "request_id", "prompt_preview", "original_sentence",
			"corrected_sentence", "entity_id", "attribute", "asserted_value",
			"authoritative_value", "policy_action", "status", "latency_ms", "payload_hash",
		})
		for _, e := range events {
			w.Write([]string{
				e.ID, e.Timestamp.Format(time.RFC3339), e.RequestID, e.PromptPreview,
				e.OriginalSentence, e.CorrectedSentence, e.EntityID, e.Attribute,
				e.AssertedValue, e.AuthoritativeValue, e.PolicyAction, e.Status,
				strconv.FormatFloat(e.LatencyMs, 'f', 2, 64), e.PayloadHash,
			})
		}
		w.Flush()
		return buf.Bytes(), nil
	}

	return json.MarshalIndent(events, "", "  ")
}

// VerifyIntegrity checks whether stored payload hashes match recomputed values.
func (l *AuditLogger) VerifyIntegrity() (int, int, error) {
	events, _, err := l.ListEvents(100000, 0, "", "")
	if err != nil {
		return 0, 0, err
	}

	valid := 0
	invalid := 0

	for _, e := range events {
		expectedHash := e.ComputePayloadHash()
		if e.PayloadHash == expectedHash {
			valid++
		} else {
			invalid++
		}
	}

	return valid, invalid, nil
}

// Close closes the audit database handle.
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.db.Close()
}
