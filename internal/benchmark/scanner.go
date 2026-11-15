package benchmark

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexandrmotologa/claimguard/internal/extractor"
	"github.com/alexandrmotologa/claimguard/internal/proxy"
	"github.com/alexandrmotologa/claimguard/internal/registry"
	"github.com/alexandrmotologa/claimguard/internal/rewriter"
	"github.com/alexandrmotologa/claimguard/internal/verifier"
)

// Scenario represents a red-teaming test case designed to provoke a hallucination.
type Scenario struct {
	ID                    string `json:"id"`
	Domain                string `json:"domain"`
	Prompt                string `json:"prompt"`
	TargetEntity          string `json:"target_entity"`
	HallucinationExpected string `json:"hallucination_expected"`
}

// ScenarioResult captures the audit outcome of a single benchmark test.
type ScenarioResult struct {
	ScenarioID     string  `json:"scenario_id"`
	Domain         string  `json:"domain"`
	Prompt         string  `json:"prompt"`
	TargetEntity   string  `json:"target_entity"`
	RawOutput      string  `json:"raw_output"`
	GuardedOutput  string  `json:"guarded_output"`
	WasIntercepted bool    `json:"was_intercepted"`
	ActionTaken    string  `json:"action_taken"`
	Status         string  `json:"status"` // PASSED, FAILED
	LatencyMs      float64 `json:"latency_ms"`
}

// Report encapsulates a complete compliance red-teaming audit certification.
type Report struct {
	Timestamp               time.Time        `json:"timestamp"`
	SuiteName               string           `json:"suite_name"`
	TotalScenarios          int              `json:"total_scenarios"`
	PassedScenarios         int              `json:"passed_scenarios"`
	ComplianceAccuracyScore float64          `json:"compliance_accuracy_score"`
	AverageLatencyMs        float64          `json:"average_latency_ms"`
	CertificationHash       string           `json:"certification_hash"`
	Scenarios               []ScenarioResult `json:"scenarios"`
}

// Runner executes automated adversarial test suites against the proxy pipeline.
type Runner struct {
	store    *registry.Store
	ext      *extractor.Extractor
	ver      *verifier.Engine
	rew      *rewriter.Rewriter
	upstream proxy.UpstreamClient
}

// NewRunner creates a new compliance benchmark runner.
func NewRunner(
	store *registry.Store,
	ext *extractor.Extractor,
	ver *verifier.Engine,
	rew *rewriter.Rewriter,
	upstream proxy.UpstreamClient,
) *Runner {
	return &Runner{
		store:    store,
		ext:      ext,
		ver:      ver,
		rew:      rew,
		upstream: upstream,
	}
}

// LoadScenarios parses scenarios from a JSONL file.
func LoadScenarios(path string) ([]Scenario, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open suite file: %w", err)
	}
	defer file.Close()

	var scenarios []Scenario
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var sc Scenario
		if err := json.Unmarshal([]byte(line), &sc); err != nil {
			continue
		}
		scenarios = append(scenarios, sc)
	}

	return scenarios, scanner.Err()
}

// RunSuite runs a set of scenarios and compiles a cryptographic compliance report.
func (r *Runner) RunSuite(ctx context.Context, suiteName string, scenarios []Scenario) (*Report, error) {
	var results []ScenarioResult
	totalLatency := 0.0
	passedCount := 0

	for _, sc := range scenarios {
		start := time.Now()

		req := &proxy.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []proxy.ChatMessage{
				{Role: "user", Content: sc.Prompt},
			},
		}

		resp, err := r.upstream.Complete(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("upstream execution for %s: %w", sc.ID, err)
		}

		rawOutput := ""
		if len(resp.Choices) > 0 {
			rawOutput = resp.Choices[0].Message.Content
		}

		// Evaluate sentences through ClaimGuard
		sentences, _ := extractor.SplitSentences(rawOutput, true)
		var guardedSentences []string
		wasIntercepted := false
		primaryAction := "verified"

		for _, s := range sentences {
			var checkResults []*verifier.VerificationResult

			// Forbidden rules check
			forbiddenRes, err := r.ver.VerifyForbidden(s.Text)
			if err == nil && len(forbiddenRes) > 0 {
				checkResults = append(checkResults, forbiddenRes...)
			}

			// Fact claims check
			claims, err := r.ext.ExtractClaims(s.Text, sc.Prompt)
			if err == nil {
				for _, c := range claims {
					res, err := r.ver.VerifyClaim(c)
					if err == nil {
						checkResults = append(checkResults, res)
					}
				}
			}

			if len(checkResults) > 0 {
				wasIntercepted = true
			}

			tRes := r.rew.Transform(s.Text, checkResults)
			if tRes.PrimaryAction != "verified" {
				primaryAction = tRes.PrimaryAction
			}

			if tRes.ShouldHalt {
				guardedSentences = append(guardedSentences, tRes.TransformedSentence)
				break
			}
			guardedSentences = append(guardedSentences, tRes.TransformedSentence)
		}

		guardedOutput := strings.Join(guardedSentences, " ")
		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		totalLatency += latencyMs

		// Scenario passes if claimguard successfully intercepted the intentional hallucination
		status := "PASSED"
		if !wasIntercepted {
			status = "FAILED"
		} else {
			passedCount++
		}

		results = append(results, ScenarioResult{
			ScenarioID:     sc.ID,
			Domain:         sc.Domain,
			Prompt:         sc.Prompt,
			TargetEntity:   sc.TargetEntity,
			RawOutput:      rawOutput,
			GuardedOutput:  guardedOutput,
			WasIntercepted: wasIntercepted,
			ActionTaken:    primaryAction,
			Status:         status,
			LatencyMs:      latencyMs,
		})
	}

	total := len(scenarios)
	score := 0.0
	avgLatency := 0.0
	if total > 0 {
		score = (float64(passedCount) / float64(total)) * 100.0
		avgLatency = totalLatency / float64(total)
	}

	report := &Report{
		Timestamp:               time.Now().UTC(),
		SuiteName:               suiteName,
		TotalScenarios:          total,
		PassedScenarios:         passedCount,
		ComplianceAccuracyScore: score,
		AverageLatencyMs:        avgLatency,
		Scenarios:               results,
	}

	// Compute certification hash
	reportBytes, _ := json.Marshal(results)
	hash := sha256.Sum256(reportBytes)
	report.CertificationHash = hex.EncodeToString(hash[:])

	return report, nil
}
