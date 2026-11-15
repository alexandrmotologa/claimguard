package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/claimguard/internal/config"
	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/rewriter"
	"github.com/alexandrmotologa/claimguard/internal/telemetry"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

// ProxyHandler handles OpenAI-compatible `/v1/chat/completions` requests.
type ProxyHandler struct {
	cfg         *config.Config
	upstream    UpstreamClient
	interceptor *StreamInterceptor
	extractor   *extractor.Extractor
	verifier    *verifier.Engine
	rewriter    *rewriter.Rewriter
	auditLogger *telemetry.AuditLogger
	metrics     *telemetry.MetricsTracker
	broadcaster *telemetry.Broadcaster
}

// NewProxyHandler initializes the HTTP proxy handler.
func NewProxyHandler(
	cfg *config.Config,
	upstream UpstreamClient,
	ext *extractor.Extractor,
	ver *verifier.Engine,
	rew *rewriter.Rewriter,
	audit *telemetry.AuditLogger,
	metrics *telemetry.MetricsTracker,
	broadcaster *telemetry.Broadcaster,
) *ProxyHandler {
	interceptor := NewStreamInterceptor(ext, ver, rew, audit, metrics, broadcaster)

	return &ProxyHandler{
		cfg:         cfg,
		upstream:    upstream,
		interceptor: interceptor,
		extractor:   ext,
		verifier:    ver,
		rewriter:    rew,
		auditLogger: audit,
		metrics:     metrics,
		broadcaster: broadcaster,
	}
}

// ServeHTTP routes `/v1/chat/completions` and `/v1/models`.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for client applications
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := strings.TrimRight(r.URL.Path, "/")
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleChatCompletions(w, r)
	case strings.HasSuffix(path, "/models"):
		h.handleModels(w, r)
	case strings.HasSuffix(path, "/metrics"):
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Write([]byte(h.metrics.PrometheusFormat()))
	default:
		http.NotFound(w, r)
	}
}

func (h *ProxyHandler) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid json body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		req.Model = h.cfg.Upstream.DefaultModel
	}

	promptContext := req.ExtractPromptText()
	requestID := fmt.Sprintf("chatcmpl-cg-%d", time.Now().UnixNano())

	// Handle streaming completion
	if req.Stream {
		upstreamStream, err := h.upstream.StreamCompletion(r.Context(), &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("upstream stream error: %v", err), http.StatusBadGateway)
			return
		}

		if err := h.interceptor.InterceptStream(r.Context(), upstreamStream, w, requestID, req.Model, promptContext); err != nil {
			// Headers already sent for SSE; connection will close
			return
		}
		return
	}

	// Handle non-streaming completion
	resp, err := h.upstream.Complete(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("upstream completion error: %v", err), http.StatusBadGateway)
		return
	}

	// Intercept and verify sentences in non-streaming response
	if len(resp.Choices) > 0 {
		rawText := resp.Choices[0].Message.Content
		sentences, _ := extractor.SplitSentences(rawText, true)

		var transformedSentences []string
		for _, s := range sentences {
			start := time.Now()
			h.metrics.RecordSentence()

			var results []*verifier.VerificationResult

			// 1. Check forbidden commitments
			forbiddenResults, err := h.verifier.VerifyForbidden(s.Text)
			if err == nil && len(forbiddenResults) > 0 {
				results = append(results, forbiddenResults...)
			}

			// 2. Check factual and numerical claims
			claims, err := h.extractor.ExtractClaims(s.Text, promptContext)
			if err == nil {
				for _, c := range claims {
					h.metrics.RecordClaimVerified()
					res, err := h.verifier.VerifyClaim(c)
					if err != nil {
						continue
					}
					results = append(results, res)
				}
			}

			if len(results) == 0 {
				transformedSentences = append(transformedSentences, s.Text)
				continue
			}

			tRes := h.rewriter.Transform(s.Text, results)
			latencyMs := float64(time.Since(start).Microseconds()) / 1000.0

			for _, res := range results {
				if res.Status == verifier.StatusContradicted {
					h.metrics.RecordContradiction(res.AppliedAction, res.Claim.EntityID, latencyMs)

					event := telemetry.AuditEvent{
						ID:                 fmt.Sprintf("audit-%d-%s", time.Now().UnixNano(), res.Claim.EntityID),
						Timestamp:          time.Now().UTC(),
						RequestID:          requestID,
						PromptPreview:      promptContext,
						OriginalSentence:   s.Text,
						CorrectedSentence:  tRes.TransformedSentence,
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

					if h.auditLogger != nil {
						_ = h.auditLogger.RecordEvent(event)
					}
					if h.broadcaster != nil {
						h.broadcaster.PublishAudit(event)
					}
				}
			}

			if tRes.ShouldHalt {
				resp.Choices[0].Message.Content = tRes.TransformedSentence
				transformedSentences = nil
				break
			}

			transformedSentences = append(transformedSentences, tRes.TransformedSentence)
		}

		if len(transformedSentences) > 0 {
			resp.Choices[0].Message.Content = strings.Join(transformedSentences, " ")
		}
	}

	h.metrics.RecordRequest(int64(len(strings.Fields(resp.Choices[0].Message.Content))))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ProxyHandler) handleModels(w http.ResponseWriter, r *http.Request) {
	modelsResponse := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"id":       "gpt-4o",
				"object":   "model",
				"created":  1715367049,
				"owned_by": "claimguard",
			},
			{
				"id":       "claimguard-guarded",
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": "claimguard",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modelsResponse)
}
