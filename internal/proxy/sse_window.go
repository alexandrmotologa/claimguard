package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/rewriter"
	"github.com/alexandrmotologa/claimguard/internal/telemetry"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

// StreamInterceptor processes streaming SSE tokens through sentence buffering, claim extraction, and active rewriting.
type StreamInterceptor struct {
	extractor   *extractor.Extractor
	verifier    *verifier.Engine
	rewriter    *rewriter.Rewriter
	auditLogger *telemetry.AuditLogger
	metrics     *telemetry.MetricsTracker
	broadcaster *telemetry.Broadcaster
}

// NewStreamInterceptor creates a new streaming token interceptor.
func NewStreamInterceptor(
	ext *extractor.Extractor,
	ver *verifier.Engine,
	rew *rewriter.Rewriter,
	audit *telemetry.AuditLogger,
	metrics *telemetry.MetricsTracker,
	broadcaster *telemetry.Broadcaster,
) *StreamInterceptor {
	return &StreamInterceptor{
		extractor:   ext,
		verifier:    ver,
		rewriter:    rew,
		auditLogger: audit,
		metrics:     metrics,
		broadcaster: broadcaster,
	}
}

// InterceptStream reads raw SSE lines from upstream, segments sentences, verifies claims, and flushes transformed chunks.
func (si *StreamInterceptor) InterceptStream(
	ctx context.Context,
	upstreamReader io.ReadCloser,
	w http.ResponseWriter,
	requestID string,
	model string,
	promptContext string,
) error {
	defer upstreamReader.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	scanner := bufio.NewScanner(upstreamReader)
	// Allocate 64KB buffer for long lines
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var buffer string
	createdTimestamp := time.Now().Unix()
	var totalTokens int64

	emitChunk := func(content string, finishReason *string) {
		chunk := StreamChunk{
			ID:      requestID,
			Object:  "chat.completion.chunk",
			Created: createdTimestamp,
			Model:   model,
			Choices: []StreamChoice{
				{
					Index: 0,
					Delta: StreamDelta{
						Content: content,
					},
					FinishReason: finishReason,
				},
			},
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		flusher.Flush()
	}

	processSentence := func(sentence string) bool {
		start := time.Now()
		si.metrics.RecordSentence()

		var results []*verifier.VerificationResult

		// 1. Check for prohibited commitments and negative invariants
		forbiddenResults, err := si.verifier.VerifyForbidden(sentence)
		if err == nil && len(forbiddenResults) > 0 {
			results = append(results, forbiddenResults...)
		}

		// 2. Check for extracted factual and numerical claims
		claims, err := si.extractor.ExtractClaims(sentence, promptContext)
		if err == nil {
			for _, c := range claims {
				si.metrics.RecordClaimVerified()
				res, err := si.verifier.VerifyClaim(c)
				if err != nil {
					continue
				}
				results = append(results, res)
			}
		}

		if len(results) == 0 {
			// No verifiable claim or forbidden pattern, release directly
			emitChunk(sentence+" ", nil)
			return true
		}

		transformRes := si.rewriter.Transform(sentence, results)
		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0

		// Audit and telemetry logging for any detected contradictions
		for _, res := range results {
			if res.Status == verifier.StatusContradicted {
				si.metrics.RecordContradiction(res.AppliedAction, res.Claim.EntityID, latencyMs)

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

				if si.auditLogger != nil {
					_ = si.auditLogger.RecordEvent(event)
				}
				if si.broadcaster != nil {
					si.broadcaster.PublishAudit(event)
				}
			}
		}

		if transformRes.ShouldHalt {
			// Emit refusal message and terminate stream
			emitChunk(transformRes.TransformedSentence, nil)
			stopReason := "stop"
			emitChunk("", &stopReason)
			fmt.Fprint(w, "data: [DONE]\n\n")
			flusher.Flush()
			return false // Signal halt
		}

		emitChunk(transformRes.TransformedSentence+" ", nil)
		return true
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
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

		var chunk StreamChunk
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

		totalTokens++
		buffer += delta

		sentences, remainder := extractor.SplitSentences(buffer, false)
		for _, s := range sentences {
			if shouldContinue := processSentence(s.Text); !shouldContinue {
				si.metrics.RecordRequest(totalTokens)
				return nil
			}
		}
		buffer = remainder
	}

	// Flush any leftover text in the buffer (EOF)
	if strings.TrimSpace(buffer) != "" {
		sentences, _ := extractor.SplitSentences(buffer, true)
		for _, s := range sentences {
			if shouldContinue := processSentence(s.Text); !shouldContinue {
				si.metrics.RecordRequest(totalTokens)
				return nil
			}
		}
	}

	si.metrics.RecordRequest(totalTokens)

	// Send final termination chunk
	stopReason := "stop"
	emitChunk("", &stopReason)
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()

	return scanner.Err()
}
