package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/claimguard/internal/config"
)

// ChatMessage represents a role/content pair in chat completion requests.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents an OpenAI-compatible completion request payload.
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// ExtractPromptText joins recent user messages to form the context for entity disambiguation.
func (r *ChatCompletionRequest) ExtractPromptText() string {
	var parts []string
	for _, m := range r.Messages {
		if m.Role == "user" {
			parts = append(parts, m.Content)
		}
	}
	return strings.Join(parts, " ")
}

// StreamDelta represents delta content inside an SSE streaming chunk.
type StreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// StreamChoice represents a candidate choice in an SSE chunk.
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason,omitempty"`
}

// StreamChunk represents a single Server-Sent Event object from the LLM.
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// NonStreamingChoice represents a choice in non-streaming responses.
type NonStreamingChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// NonStreamingResponse represents a full JSON completion response.
type NonStreamingResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []NonStreamingChoice `json:"choices"`
}

// UpstreamClient defines the interface for interacting with upstream LLM engines.
type UpstreamClient interface {
	StreamCompletion(ctx context.Context, req *ChatCompletionRequest) (io.ReadCloser, error)
	Complete(ctx context.Context, req *ChatCompletionRequest) (*NonStreamingResponse, error)
}

// HTTPUpstreamClient calls live external providers (OpenAI, Ollama, Anthropic proxy).
type HTTPUpstreamClient struct {
	cfg        config.UpstreamConfig
	httpClient *http.Client
}

// NewHTTPUpstreamClient creates an HTTP client for external LLMs.
func NewHTTPUpstreamClient(cfg config.UpstreamConfig) *HTTPUpstreamClient {
	return &HTTPUpstreamClient{
		cfg:        cfg,
		httpClient: &http.Client{},
	}
}

// StreamCompletion sends an SSE streaming request to the upstream LLM.
func (c *HTTPUpstreamClient) StreamCompletion(ctx context.Context, req *ChatCompletionRequest) (io.ReadCloser, error) {
	req.Stream = true
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("upstream returned error %d: %s", resp.StatusCode, string(errBody))
	}

	return resp.Body, nil
}

// Complete executes a non-streaming completion request.
func (c *HTTPUpstreamClient) Complete(ctx context.Context, req *ChatCompletionRequest) (*NonStreamingResponse, error) {
	req.Stream = false
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream returned error %d: %s", resp.StatusCode, string(errBody))
	}

	var completionResp NonStreamingResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return nil, fmt.Errorf("decode upstream response: %w", err)
	}

	return &completionResp, nil
}

// FailoverClient wraps a primary provider and one or more backup providers.
type FailoverClient struct {
	primary   UpstreamClient
	fallbacks []UpstreamClient
}

// NewFailoverClient creates an upstream client with automatic multi-model failover.
func NewFailoverClient(primary UpstreamClient, fallbacks ...UpstreamClient) *FailoverClient {
	return &FailoverClient{
		primary:   primary,
		fallbacks: fallbacks,
	}
}

// StreamCompletion attempts primary streaming, falling back to backup models on failure.
func (fc *FailoverClient) StreamCompletion(ctx context.Context, req *ChatCompletionRequest) (io.ReadCloser, error) {
	stream, err := fc.primary.StreamCompletion(ctx, req)
	if err == nil {
		return stream, nil
	}

	for _, fb := range fc.fallbacks {
		stream, fbErr := fb.StreamCompletion(ctx, req)
		if fbErr == nil {
			return stream, nil
		}
	}

	return nil, fmt.Errorf("all primary and fallback providers failed: %w", err)
}

// Complete attempts primary completion, falling back to backup models on failure.
func (fc *FailoverClient) Complete(ctx context.Context, req *ChatCompletionRequest) (*NonStreamingResponse, error) {
	resp, err := fc.primary.Complete(ctx, req)
	if err == nil {
		return resp, nil
	}

	for _, fb := range fc.fallbacks {
		resp, fbErr := fb.Complete(ctx, req)
		if fbErr == nil {
			return resp, nil
		}
	}

	return nil, fmt.Errorf("all primary and fallback providers failed: %w", err)
}

// BuildUpstreamClient creates an UpstreamClient with optional configured fallbacks.
func BuildUpstreamClient(cfg config.UpstreamConfig) UpstreamClient {
	var primary UpstreamClient
	if cfg.Provider == "simulator" {
		primary = NewSimulator(15 * time.Millisecond)
	} else {
		primary = NewHTTPUpstreamClient(cfg)
	}

	if len(cfg.Fallbacks) == 0 {
		return primary
	}

	var fallbacks []UpstreamClient
	for _, fbCfg := range cfg.Fallbacks {
		if fbCfg.Provider == "simulator" {
			fallbacks = append(fallbacks, NewSimulator(15*time.Millisecond))
		} else {
			fallbacks = append(fallbacks, NewHTTPUpstreamClient(fbCfg))
		}
	}

	return NewFailoverClient(primary, fallbacks...)
}
