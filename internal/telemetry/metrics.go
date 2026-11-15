package telemetry

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsTracker aggregates runtime performance and hallucination interception statistics.
type MetricsTracker struct {
	mu sync.RWMutex

	totalRequests         atomic.Int64
	totalTokens           atomic.Int64
	totalSentences        atomic.Int64
	totalClaimsVerified   atomic.Int64
	totalContradictions   atomic.Int64

	actionCounts          map[string]*atomic.Int64
	entityViolations      map[string]*atomic.Int64
	recentLatencies       []float64
	maxRecentLatencies    int
	startTime             time.Time
}

// MetricsSnapshot provides a point-in-time view of telemetry indicators.
type MetricsSnapshot struct {
	UptimeSeconds          float64          `json:"uptime_seconds"`
	TotalRequests          int64            `json:"total_requests"`
	TotalTokens            int64            `json:"total_tokens"`
	TotalSentences         int64            `json:"total_sentences"`
	TotalClaimsVerified    int64            `json:"total_claims_verified"`
	TotalContradictions    int64            `json:"total_contradictions"`
	InterceptionRatePct    float64          `json:"interception_rate_pct"`
	AverageLatencyMs       float64          `json:"average_latency_ms"`
	InterventionsByAction  map[string]int64 `json:"interventions_by_action"`
	ViolationsByEntity     map[string]int64 `json:"violations_by_entity"`
}

// NewMetricsTracker initializes a new thread-safe metrics collector.
func NewMetricsTracker() *MetricsTracker {
	actions := map[string]*atomic.Int64{
		"rewrite":     new(atomic.Int64),
		"disclaimer":  new(atomic.Int64),
		"strict_halt": new(atomic.Int64),
		"flag_only":   new(atomic.Int64),
	}

	return &MetricsTracker{
		actionCounts:       actions,
		entityViolations:   make(map[string]*atomic.Int64),
		recentLatencies:    make([]float64, 0, 100),
		maxRecentLatencies: 100,
		startTime:          time.Now(),
	}
}

// RecordRequest increments request and token counters.
func (m *MetricsTracker) RecordRequest(tokens int64) {
	m.totalRequests.Add(1)
	m.totalTokens.Add(tokens)
}

// RecordSentence increments sentence evaluation counter.
func (m *MetricsTracker) RecordSentence() {
	m.totalSentences.Add(1)
}

// RecordClaimVerified increments verified claim counter.
func (m *MetricsTracker) RecordClaimVerified() {
	m.totalClaimsVerified.Add(1)
}

// RecordContradiction records an interception event with its action and entity.
func (m *MetricsTracker) RecordContradiction(action, entityID string, latencyMs float64) {
	m.totalContradictions.Add(1)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Track action
	if counter, ok := m.actionCounts[action]; ok {
		counter.Add(1)
	} else {
		newCounter := new(atomic.Int64)
		newCounter.Store(1)
		m.actionCounts[action] = newCounter
	}

	// Track entity violation
	if entityID != "" {
		if counter, ok := m.entityViolations[entityID]; ok {
			counter.Add(1)
		} else {
			newCounter := new(atomic.Int64)
			newCounter.Store(1)
			m.entityViolations[entityID] = newCounter
		}
	}

	// Track latency sample
	if len(m.recentLatencies) >= m.maxRecentLatencies {
		m.recentLatencies = m.recentLatencies[1:]
	}
	m.recentLatencies = append(m.recentLatencies, latencyMs)
}

// Snapshot returns the current aggregated telemetry state.
func (m *MetricsTracker) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	actions := make(map[string]int64)
	for k, v := range m.actionCounts {
		actions[k] = v.Load()
	}

	violations := make(map[string]int64)
	for k, v := range m.entityViolations {
		violations[k] = v.Load()
	}

	avgLatency := 0.0
	if len(m.recentLatencies) > 0 {
		sum := 0.0
		for _, l := range m.recentLatencies {
			sum += l
		}
		avgLatency = sum / float64(len(m.recentLatencies))
	}

	totalClaims := m.totalClaimsVerified.Load()
	totalInterceptions := m.totalContradictions.Load()
	interceptionRate := 0.0
	if totalClaims > 0 {
		interceptionRate = (float64(totalInterceptions) / float64(totalClaims)) * 100.0
	}

	return MetricsSnapshot{
		UptimeSeconds:         time.Since(m.startTime).Seconds(),
		TotalRequests:         m.totalRequests.Load(),
		TotalTokens:           m.totalTokens.Load(),
		TotalSentences:        m.totalSentences.Load(),
		TotalClaimsVerified:   totalClaims,
		TotalContradictions:   totalInterceptions,
		InterceptionRatePct:   interceptionRate,
		AverageLatencyMs:      avgLatency,
		InterventionsByAction: actions,
		ViolationsByEntity:    violations,
	}
}

// PrometheusFormat serializes current telemetry metrics into standard OpenMetrics / Prometheus exposition text.
func (m *MetricsTracker) PrometheusFormat() string {
	snap := m.Snapshot()

	var b strings.Builder
	b.WriteString("# HELP claimguard_uptime_seconds Total runtime in seconds\n")
	b.WriteString("# TYPE claimguard_uptime_seconds gauge\n")
	fmt.Fprintf(&b, "claimguard_uptime_seconds %.2f\n\n", snap.UptimeSeconds)

	b.WriteString("# HELP claimguard_requests_total Total number of chat completion requests\n")
	b.WriteString("# TYPE claimguard_requests_total counter\n")
	fmt.Fprintf(&b, "claimguard_requests_total %d\n\n", snap.TotalRequests)

	b.WriteString("# HELP claimguard_tokens_scanned_total Total number of tokens scanned\n")
	b.WriteString("# TYPE claimguard_tokens_scanned_total counter\n")
	fmt.Fprintf(&b, "claimguard_tokens_scanned_total %d\n\n", snap.TotalTokens)

	b.WriteString("# HELP claimguard_sentences_analyzed_total Total number of sentences analyzed\n")
	b.WriteString("# TYPE claimguard_sentences_analyzed_total counter\n")
	fmt.Fprintf(&b, "claimguard_sentences_analyzed_total %d\n\n", snap.TotalSentences)

	b.WriteString("# HELP claimguard_claims_verified_total Total claims cross-referenced against ground truth\n")
	b.WriteString("# TYPE claimguard_claims_verified_total counter\n")
	fmt.Fprintf(&b, "claimguard_claims_verified_total %d\n\n", snap.TotalClaimsVerified)

	b.WriteString("# HELP claimguard_contradictions_total Total intercepted factual contradictions\n")
	b.WriteString("# TYPE claimguard_contradictions_total counter\n")
	fmt.Fprintf(&b, "claimguard_contradictions_total %d\n\n", snap.TotalContradictions)

	b.WriteString("# HELP claimguard_interception_rate_percent Intercepted contradictions percentage\n")
	b.WriteString("# TYPE claimguard_interception_rate_percent gauge\n")
	fmt.Fprintf(&b, "claimguard_interception_rate_percent %.2f\n\n", snap.InterceptionRatePct)

	b.WriteString("# HELP claimguard_average_latency_ms Average sentence interception latency in milliseconds\n")
	b.WriteString("# TYPE claimguard_average_latency_ms gauge\n")
	fmt.Fprintf(&b, "claimguard_average_latency_ms %.4f\n\n", snap.AverageLatencyMs)

	b.WriteString("# HELP claimguard_interventions_total Interventions by policy action\n")
	b.WriteString("# TYPE claimguard_interventions_total counter\n")
	for action, count := range snap.InterventionsByAction {
		fmt.Fprintf(&b, "claimguard_interventions_total{action=\"%s\"} %d\n", action, count)
	}
	b.WriteString("\n")

	b.WriteString("# HELP claimguard_violations_by_entity_total Violations grouped by entity\n")
	b.WriteString("# TYPE claimguard_violations_by_entity_total counter\n")
	for entity, count := range snap.ViolationsByEntity {
		fmt.Fprintf(&b, "claimguard_violations_by_entity_total{entity_id=\"%s\"} %d\n", entity, count)
	}

	return b.String()
}
