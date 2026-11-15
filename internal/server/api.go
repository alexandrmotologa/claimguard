package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/claimguard/internal/benchmark"
	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/proxy"
	"github.com/alexandrmotologa/claimguard/internal/registry"
	"github.com/alexandrmotologa/claimguard/internal/rewriter"
	"github.com/alexandrmotologa/claimguard/internal/telemetry"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

// APIServer serves REST endpoints for the verification studio.
type APIServer struct {
	store        *registry.Store
	verifier     *verifier.Engine
	rewriter     *rewriter.Rewriter
	extractor    *extractor.Extractor
	auditLogger  *telemetry.AuditLogger
	metrics      *telemetry.MetricsTracker
	broadcaster  *Broadcaster
	cfg          *config.Config
	upstream     proxy.UpstreamClient
	latestReport *benchmark.Report
	reportMu     sync.RWMutex
}

// NewAPIServer creates an API server handler for studio controls and telemetry.
func NewAPIServer(
	store *registry.Store,
	ver *verifier.Engine,
	rew *rewriter.Rewriter,
	ext *extractor.Extractor,
	audit *telemetry.AuditLogger,
	metrics *telemetry.MetricsTracker,
	broadcaster *Broadcaster,
	cfg *config.Config,
	upstream proxy.UpstreamClient,
) *APIServer {
	return &APIServer{
		store:       store,
		verifier:    ver,
		rewriter:    rew,
		extractor:   ext,
		auditLogger: audit,
		metrics:     metrics,
		broadcaster: broadcaster,
		cfg:         cfg,
		upstream:    upstream,
	}
}

// RegisterRoutes registers all API endpoints on an HTTP mux.
func (s *APIServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/metrics", s.handlePrometheus)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/entities", s.handleEntities)
	mux.HandleFunc("/api/entities/", s.handleEntityDetail)
	mux.HandleFunc("/api/facts", s.handleFacts)
	mux.HandleFunc("/api/facts/", s.handleFactDetail)
	mux.HandleFunc("/api/audit", s.handleAudit)
	mux.HandleFunc("/api/audit/export", s.handleAuditExport)
	mux.HandleFunc("/api/policy", s.handlePolicy)
	mux.HandleFunc("/api/forbidden", s.handleForbidden)
	mux.HandleFunc("/api/forbidden/", s.handleForbiddenDetail)
	mux.HandleFunc("/api/benchmark/run", s.handleBenchmarkRun)
	mux.HandleFunc("/api/benchmark/latest", s.handleBenchmarkLatest)
	mux.HandleFunc("/api/verify/dry-run", s.handleDryRun)
	mux.HandleFunc("/api/stream/compare", s.handleStreamCompare)
	mux.HandleFunc("/api/events/live", s.broadcaster.ServeHTTP)
}

func (s *APIServer) handlePrometheus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(s.metrics.PrometheusFormat()))
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "claimguard",
	})
}

func (s *APIServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	snap := s.metrics.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap)
}

func (s *APIServer) handleEntities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		entities, err := s.store.ListEntities()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(entities)
	case http.MethodPost:
		var e registry.Entity
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if e.ID == "" || e.Name == "" {
			http.Error(w, "entity id and name are required", http.StatusBadRequest)
			return
		}
		if err := s.store.SaveEntity(e); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(e)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleEntityDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/entities/")
	if id == "" {
		http.Error(w, "missing entity id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		e, err := s.store.GetEntity(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if e == nil {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(e)
	case http.MethodDelete:
		if err := s.store.DeleteEntity(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleFacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		entityID := r.URL.Query().Get("entity_id")
		var facts []registry.Fact
		var err error
		if entityID != "" {
			facts, err = s.store.FindFactsByEntity(entityID)
		} else {
			facts, err = s.store.ListFacts()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(facts)
	case http.MethodPost:
		var f registry.Fact
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if f.ID == "" || f.EntityID == "" || f.Attribute == "" {
			http.Error(w, "fact id, entity_id, and attribute are required", http.StatusBadRequest)
			return
		}
		if err := s.store.SaveFact(f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(f)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleFactDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/facts/")
	if id == "" {
		http.Error(w, "missing fact id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		f, err := s.store.GetFact(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if f == nil {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(f)
	case http.MethodPut:
		var f registry.Fact
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		f.ID = id
		if err := s.store.SaveFact(f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(f)
	case http.MethodDelete:
		if err := s.store.DeleteFact(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleAudit(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	statusFilter := r.URL.Query().Get("status")
	entityFilter := r.URL.Query().Get("entity_id")

	limit := 50
	offset := 0
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	events, total, err := s.auditLogger.ListEvents(limit, offset, statusFilter, entityFilter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (s *APIServer) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	data, err := s.auditLogger.ExportEvents(format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=claimguard_audit.csv")
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=claimguard_audit.json")
	}

	w.Write(data)
}

func (s *APIServer) handlePolicy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(s.verifier.GetPolicy())
	case http.MethodPut:
		var p config.PolicyConfig
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.verifier.UpdatePolicy(p)
		s.rewriter.UpdatePolicy(p)
		json.NewEncoder(w).Encode(p)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleForbidden(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		patterns, err := s.store.ListForbiddenPatterns()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(patterns)
	case http.MethodPost:
		var fp registry.ForbiddenPattern
		if err := json.NewDecoder(r.Body).Decode(&fp); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if fp.ID == "" || fp.Pattern == "" {
			http.Error(w, "pattern id and regex pattern are required", http.StatusBadRequest)
			return
		}
		if err := s.store.SaveForbiddenPattern(fp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(fp)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleForbiddenDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/forbidden/")
	if id == "" {
		http.Error(w, "missing pattern id", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		fp, err := s.store.GetForbiddenPattern(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if fp == nil {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(fp)
	case http.MethodDelete:
		if err := s.store.DeleteForbiddenPattern(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleBenchmarkRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	suitePath := "examples/test_queries.jsonl"
	type RunReq struct {
		SuitePath string `json:"suite_path"`
	}
	var req RunReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.SuitePath != "" {
		suitePath = req.SuitePath
	}

	scenarios, err := benchmark.LoadScenarios(suitePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	runner := benchmark.NewRunner(s.store, s.extractor, s.verifier, s.rewriter, s.upstream)
	report, err := runner.RunSuite(r.Context(), "Automated Red-Teaming Compliance Suite", scenarios)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.reportMu.Lock()
	s.latestReport = report
	s.reportMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (s *APIServer) handleBenchmarkLatest(w http.ResponseWriter, r *http.Request) {
	s.reportMu.RLock()
	report := s.latestReport
	s.reportMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if report == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "no_runs_yet",
			"message": "Click 'Run Red-Team Scan' to execute automated compliance suite",
		})
		return
	}

	json.NewEncoder(w).Encode(report)
}

// DryRunRequest allows testing arbitrary sentences directly from the verification studio.
type DryRunRequest struct {
	Sentence      string `json:"sentence"`
	PromptContext string `json:"prompt_context"`
}

// DryRunResponse encapsulates dry-run extraction, verification, and transformation.
type DryRunResponse struct {
	OriginalSentence    string                         `json:"original_sentence"`
	Claims              []extractor.ExtractedClaim     `json:"claims"`
	Results             []*verifier.VerificationResult `json:"results"`
	TransformedSentence string                         `json:"transformed_sentence"`
	PrimaryAction       string                         `json:"primary_action"`
	ShouldHalt          bool                           `json:"should_halt"`
}

func (s *APIServer) handleDryRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DryRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var results []*verifier.VerificationResult

	// 1. Check forbidden commitments
	forbiddenResults, err := s.verifier.VerifyForbidden(req.Sentence)
	if err == nil && len(forbiddenResults) > 0 {
		results = append(results, forbiddenResults...)
	}

	// 2. Check factual and numerical claims
	claims, err := s.extractor.ExtractClaims(req.Sentence, req.PromptContext)
	if err == nil {
		for _, c := range claims {
			res, err := s.verifier.VerifyClaim(c)
			if err != nil {
				continue
			}
			results = append(results, res)
		}
	}

	transformRes := s.rewriter.Transform(req.Sentence, results)

	resp := DryRunResponse{
		OriginalSentence:    req.Sentence,
		Claims:              claims,
		Results:             results,
		TransformedSentence: transformRes.TransformedSentence,
		PrimaryAction:       transformRes.PrimaryAction,
		ShouldHalt:          transformRes.ShouldHalt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// StreamCompareRequest holds the prompt and optional model for side-by-side stream inspection.
type StreamCompareRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"`
}

func (s *APIServer) handleStreamCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StreamCompareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	model := req.Model
	if model == "" {
		model = s.cfg.Upstream.DefaultModel
	}

	chatReq := &proxy.ChatCompletionRequest{
		Model: model,
		Messages: []proxy.ChatMessage{
			{Role: "user", Content: req.Prompt},
		},
		Stream: true,
	}

	upstreamStream, err := s.upstream.StreamCompletion(r.Context(), chatReq)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer upstreamStream.Close()

	scanner := bufio.NewScanner(upstreamStream)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var buffer string
	promptContext := req.Prompt
	requestID := fmt.Sprintf("cmp-%d", time.Now().UnixNano())

	emitEvent := func(eventType string, data interface{}) {
		b, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(b))
		flusher.Flush()
	}

	processSentence := func(sentence string) {
		start := time.Now()
		s.metrics.RecordSentence()

		var results []*verifier.VerificationResult

		// Check forbidden invariants
		forbiddenResults, err := s.verifier.VerifyForbidden(sentence)
		if err == nil && len(forbiddenResults) > 0 {
			results = append(results, forbiddenResults...)
		}

		// Extract claims
		claims, err := s.extractor.ExtractClaims(sentence, promptContext)
		if err == nil {
			for _, c := range claims {
				s.metrics.RecordClaimVerified()
				res, err := s.verifier.VerifyClaim(c)
				if err != nil {
					continue
				}
				results = append(results, res)
			}
		}

		transformRes := s.rewriter.Transform(sentence, results)
		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0

		for _, res := range results {
			if res.Status == verifier.StatusContradicted {
				s.metrics.RecordContradiction(res.AppliedAction, res.Claim.EntityID, latencyMs)
				event := telemetry.AuditEvent{
					ID:                 fmt.Sprintf("audit-%d-%s", time.Now().UnixNano(), res.Claim.EntityID),
					Timestamp:          time.Now().UTC(),
					RequestID:          requestID,
					PromptPreview:      promptContext,
					OriginalSentence:   sentence,
					CorrectedSentence:  transformRes.TransformedSentence,
					EntityID:           res.Claim.EntityID,
					Attribute:          res.Claim.Attribute,
					AssertedValue:      res.Claim.AssertedValue,
					AuthoritativeValue: "",
					PolicyAction:       res.AppliedAction,
					Status:             string(res.Status),
					LatencyMs:          latencyMs,
				}
				if res.AuthoritativeFact != nil {
					event.AuthoritativeValue = res.AuthoritativeFact.ExpectedValue
				}
				if s.auditLogger != nil {
					_ = s.auditLogger.RecordEvent(event)
				}
				if s.broadcaster != nil {
					s.broadcaster.PublishAudit(event)
				}
			}
		}

		emitEvent("guarded", map[string]interface{}{
			"raw_sentence":     sentence,
			"guarded_sentence": transformRes.TransformedSentence,
			"results":          results,
			"primary_action":   transformRes.PrimaryAction,
			"should_halt":      transformRes.ShouldHalt,
		})
	}

	for scanner.Scan() {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if payload == "[DONE]" {
			break
		}

		var chunk proxy.StreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}

		// Emit raw token delta directly to client
		emitEvent("raw", map[string]string{"delta": delta})

		buffer += delta
		sentences, remainder := extractor.SplitSentences(buffer, false)
		for _, s := range sentences {
			processSentence(s.Text)
		}
		buffer = remainder
	}

	if strings.TrimSpace(buffer) != "" {
		sentences, _ := extractor.SplitSentences(buffer, true)
		for _, s := range sentences {
			processSentence(s.Text)
		}
	}

	emitEvent("done", map[string]bool{"done": true})
}
