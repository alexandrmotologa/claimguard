package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/registry"
	"github.com/alexandrmotologa/claimguard/internal/rewriter"
	"github.com/alexandrmotologa/claimguard/internal/telemetry"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

// APIServer serves REST endpoints for the verification studio.
type APIServer struct {
	store       *registry.Store
	verifier    *verifier.Engine
	rewriter    *rewriter.Rewriter
	extractor   *extractor.Extractor
	auditLogger *telemetry.AuditLogger
	metrics     *telemetry.MetricsTracker
	broadcaster *Broadcaster
	cfg         *config.Config
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
	}
}

// RegisterRoutes registers all API endpoints on an HTTP mux.
func (s *APIServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/entities", s.handleEntities)
	mux.HandleFunc("/api/entities/", s.handleEntityDetail)
	mux.HandleFunc("/api/facts", s.handleFacts)
	mux.HandleFunc("/api/facts/", s.handleFactDetail)
	mux.HandleFunc("/api/audit", s.handleAudit)
	mux.HandleFunc("/api/audit/export", s.handleAuditExport)
	mux.HandleFunc("/api/policy", s.handlePolicy)
	mux.HandleFunc("/api/verify/dry-run", s.handleDryRun)
	mux.HandleFunc("/api/events/live", s.broadcaster.ServeHTTP)
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

	claims, err := s.extractor.ExtractClaims(req.Sentence, req.PromptContext)
	if err != nil {
		http.Error(w, fmt.Sprintf("extract claims: %v", err), http.StatusInternalServerError)
		return
	}

	var results []*verifier.VerificationResult
	for _, c := range claims {
		res, err := s.verifier.VerifyClaim(c)
		if err != nil {
			continue
		}
		results = append(results, res)
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
