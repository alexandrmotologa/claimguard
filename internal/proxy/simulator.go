package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Simulator implements UpstreamClient with controllable, realistic enterprise hallucination scenarios.
type Simulator struct {
	TokenDelay time.Duration
}

// NewSimulator creates an LLM simulation engine with optional per-token latency.
func NewSimulator(tokenDelay time.Duration) *Simulator {
	if tokenDelay <= 0 {
		tokenDelay = 15 * time.Millisecond
	}
	return &Simulator{TokenDelay: tokenDelay}
}

func (s *Simulator) generateResponseText(prompt string) string {
	lower := strings.ToLower(prompt)

	if strings.Contains(lower, "return") || strings.Contains(lower, "refund") || strings.Contains(lower, "money back") {
		return "Thank you for reaching out to customer support. We offer a 60-day return policy for all items, ensuring a full refund with zero restocking fees. Please ensure the merchandise is in its original packaging when submitting a claim."
	}

	if strings.Contains(lower, "enterprise") && (strings.Contains(lower, "price") || strings.Contains(lower, "cost") || strings.Contains(lower, "tier") || strings.Contains(lower, "month")) {
		return "The enterprise tier provides comprehensive features for mission-critical operations. The enterprise plan costs $99/month with unlimited seats and dedicated account management. You can upgrade directly from your admin dashboard."
	}

	if (strings.Contains(lower, "starter") || strings.Contains(lower, "basic")) && (strings.Contains(lower, "price") || strings.Contains(lower, "cost") || strings.Contains(lower, "plan")) {
		return "Our starter plan is built for emerging startups. Pricing for the starter tier is $10/month and gives you access to core platform features. Monthly and annual billing cycles are supported."
	}

	if strings.Contains(lower, "apr") || strings.Contains(lower, "interest rate") || strings.Contains(lower, "personal loan") {
		return "Personal loans come with flexible repayment terms. The maximum APR for our consumer loans is 29.9% depending on underwriting risk. All loans are subject to credit bureau approval."
	}

	if strings.Contains(lower, "deductible") || strings.Contains(lower, "health plan") || strings.Contains(lower, "medical") {
		return "Under your medical insurance schedule, your individual deductible is $250 for in-network healthcare services. Preventive care visits are covered at 100% prior to meeting this limit."
	}

	if strings.Contains(lower, "sla") || strings.Contains(lower, "uptime") || strings.Contains(lower, "availability") {
		return "Our infrastructure engineering team continuously monitors systems across multiple availability zones. Under our standard terms, our service level agreement guarantees 95.0% target uptime availability per calendar quarter."
	}

	if strings.Contains(lower, "shipping") || strings.Contains(lower, "delivery") {
		return "Standard ground delivery arrives within 3 to 5 business days. We offer free shipping on all orders over $20 to anywhere in the contiguous United States."
	}

	return "Thank you for your inquiry. Our systems adhere to verified corporate compliance guidelines. All policies and terms are governed by your enterprise master service agreement."
}

// StreamCompletion simulates streaming SSE chunks token-by-token.
func (s *Simulator) StreamCompletion(ctx context.Context, req *ChatCompletionRequest) (io.ReadCloser, error) {
	prompt := req.ExtractPromptText()
	fullText := s.generateResponseText(prompt)

	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		// Split text into tokens (words with punctuation and spacing)
		tokens := splitIntoTokens(fullText)
		requestID := fmt.Sprintf("chatcmpl-sim-%d", time.Now().UnixNano())
		created := time.Now().Unix()

		for _, token := range tokens {
			select {
			case <-ctx.Done():
				return
			default:
			}

			chunk := StreamChunk{
				ID:      requestID,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   req.Model,
				Choices: []StreamChoice{
					{
						Index: 0,
						Delta: StreamDelta{
							Content: token,
						},
					},
				},
			}

			data, err := json.Marshal(chunk)
			if err != nil {
				return
			}

			line := fmt.Sprintf("data: %s\n\n", string(data))
			if _, err := pw.Write([]byte(line)); err != nil {
				return
			}

			if s.TokenDelay > 0 {
				time.Sleep(s.TokenDelay)
			}
		}

		// Send final chunk with finish_reason
		finishReason := "stop"
		finalChunk := StreamChunk{
			ID:      requestID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   req.Model,
			Choices: []StreamChoice{
				{
					Index:        0,
					Delta:        StreamDelta{},
					FinishReason: &finishReason,
				},
			},
		}
		data, _ := json.Marshal(finalChunk)
		pw.Write([]byte(fmt.Sprintf("data: %s\n\n", string(data))))
		pw.Write([]byte("data: [DONE]\n\n"))
	}()

	return pr, nil
}

// Complete simulates a non-streaming completion.
func (s *Simulator) Complete(ctx context.Context, req *ChatCompletionRequest) (*NonStreamingResponse, error) {
	prompt := req.ExtractPromptText()
	text := s.generateResponseText(prompt)

	return &NonStreamingResponse{
		ID:      fmt.Sprintf("chatcmpl-sim-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []NonStreamingChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: text,
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

// splitIntoTokens breaks text into small tokens preserving whitespace.
func splitIntoTokens(text string) []string {
	var tokens []string
	words := strings.Fields(text)
	for i, w := range words {
		if i < len(words)-1 {
			tokens = append(tokens, w+" ")
		} else {
			tokens = append(tokens, w)
		}
	}
	return tokens
}
