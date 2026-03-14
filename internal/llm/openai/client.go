package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/llm"
)

const (
	ProviderOpenAI   = "openai"
	ProviderDeepSeek = "deepseek"

	defaultOpenAIBaseURL   = "https://api.openai.com/v1"
	defaultDeepSeekBaseURL = "https://api.deepseek.com"
	defaultRequestTimeout  = 120 * time.Second
)

// Client implements llm.LLMProvider for OpenAI-compatible APIs.
type Client struct {
	kind       string
	baseURL    string
	apiKey     string
	model      string
	primary    string
	secondary  string
	httpClient *http.Client
	mu         sync.RWMutex
}

func NewClient(kind, baseURL, apiKey, model string, requestTimeout time.Duration) *Client {
	kind = normalizeProvider(kind)
	if baseURL == "" {
		baseURL = defaultBaseURL(kind)
	}
	if requestTimeout <= 0 {
		requestTimeout = defaultRequestTimeout
	}
	return &Client{
		kind:    kind,
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(model),
		primary: strings.TrimSpace(model),
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

func (c *Client) Name() string {
	if c == nil {
		return ""
	}
	return c.kind
}

func (c *Client) Generate(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("provider not configured")
	}
	if !c.IsAvailable(ctx) {
		return nil, fmt.Errorf("%s: missing API key", c.kind)
	}

	model := c.resolveModel(req)
	if prefersResponsesAPI(c.kind, model) {
		return c.generateResponses(ctx, model, req)
	}
	return c.generateChatCompletions(ctx, model, req)
}

func (c *Client) GenerateStream(context.Context, llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	return nil, fmt.Errorf("%s: streaming is not enabled", c.kind)
}

func (c *Client) IsAvailable(context.Context) bool {
	return c != nil && strings.TrimSpace(c.apiKey) != ""
}

func (c *Client) ModelInfo(context.Context) (*llm.ModelInfo, error) {
	model := c.resolveModel(llm.GenerateRequest{Role: llm.RolePrimary})
	if model == "" {
		return nil, fmt.Errorf("%s: model not configured", c.kind)
	}
	return &llm.ModelInfo{
		Name:     model,
		Provider: c.kind,
	}, nil
}

func (c *Client) ListModels(context.Context) ([]llm.ModelInfo, error) {
	// Cloud providers rely on configured model names instead of interactive
	// model discovery inside the TUI.
	var models []llm.ModelInfo
	if name := strings.TrimSpace(c.primary); name != "" {
		models = append(models, llm.ModelInfo{Name: name, Provider: c.kind})
	}
	if name := strings.TrimSpace(c.secondary); name != "" && name != c.primary {
		models = append(models, llm.ModelInfo{Name: name, Provider: c.kind})
	}
	return models, nil
}

func (c *Client) SetModel(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	c.mu.Lock()
	c.model = name
	c.primary = name
	c.mu.Unlock()
}

func (c *Client) SetModelForRole(role llm.ModelRole, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	c.mu.Lock()
	switch role {
	case llm.RoleSecondary:
		c.secondary = name
	default:
		c.primary = name
		c.model = name
	}
	c.mu.Unlock()
}

func (c *Client) resolveModel(req llm.GenerateRequest) string {
	if strings.TrimSpace(req.Model) != "" {
		return strings.TrimSpace(req.Model)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	switch req.Role {
	case llm.RoleSecondary:
		if strings.TrimSpace(c.secondary) != "" {
			return c.secondary
		}
	}
	if strings.TrimSpace(c.primary) != "" {
		return c.primary
	}
	return c.model
}

func (c *Client) generateChatCompletions(ctx context.Context, model string, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
	body := chatCompletionRequest{
		Model: model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.Prompt},
		},
		Temperature: req.Temperature,
	}
	raw, err := c.doJSON(ctx, "/chat/completions", body)
	if err != nil {
		return nil, err
	}

	var resp chatCompletionResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("%s: decode chat completion response: %w", c.kind, err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("%s: empty chat completion response", c.kind)
	}

	text := strings.TrimSpace(parseMessageText(resp.Choices[0].Message.Content))
	thinking := strings.TrimSpace(resp.Choices[0].Message.ReasoningContent)
	return &llm.GenerateResponse{
		Text:       text,
		Thinking:   thinking,
		Raw:        string(raw),
		TokenCount: resp.Usage.TotalTokens,
	}, nil
}

func (c *Client) generateResponses(ctx context.Context, model string, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
	input := make([]responsesInputItem, 0, 2)
	if system := strings.TrimSpace(req.System); system != "" {
		input = append(input, responsesInputItem{
			Role: "developer",
			Content: []responsesInputContent{{
				Type: "input_text",
				Text: system,
			}},
		})
	}
	input = append(input, responsesInputItem{
		Role: "user",
		Content: []responsesInputContent{{
			Type: "input_text",
			Text: req.Prompt,
		}},
	})

	body := responsesRequest{
		Model: model,
		Input: input,
		Reasoning: &responsesReasoning{
			Summary: "auto",
		},
	}
	raw, err := c.doJSON(ctx, "/responses", body)
	if err != nil {
		return nil, err
	}

	var resp responsesResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("%s: decode responses API payload: %w", c.kind, err)
	}

	text := strings.TrimSpace(resp.OutputText)
	var thinkingParts []string
	if text == "" || len(resp.Output) > 0 {
		for _, item := range resp.Output {
			switch item.Type {
			case "message":
				if text == "" {
					text = strings.TrimSpace(extractResponsesMessageText(item.Content))
				}
			case "reasoning":
				if summary := strings.TrimSpace(extractReasoningSummary(item.Summary)); summary != "" {
					thinkingParts = append(thinkingParts, summary)
				}
			}
		}
	}

	return &llm.GenerateResponse{
		Text:       text,
		Thinking:   strings.TrimSpace(strings.Join(thinkingParts, "\n\n")),
		Raw:        string(raw),
		TokenCount: resp.Usage.TotalTokens,
	}, nil
}

func (c *Client) doJSON(ctx context.Context, path string, payload any) ([]byte, error) {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid url: %w", c.kind, err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", c.kind, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", c.kind, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: request failed: %w", c.kind, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: read response: %w", c.kind, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: status %d: %s", c.kind, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

func normalizeProvider(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case ProviderDeepSeek:
		return ProviderDeepSeek
	default:
		return ProviderOpenAI
	}
}

func defaultBaseURL(kind string) string {
	switch normalizeProvider(kind) {
	case ProviderDeepSeek:
		return defaultDeepSeekBaseURL
	default:
		return defaultOpenAIBaseURL
	}
}

func prefersResponsesAPI(kind, model string) bool {
	if normalizeProvider(kind) != ProviderOpenAI {
		return false
	}
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "o") || strings.HasPrefix(model, "gpt-5") || strings.Contains(model, "reason")
}

func parseMessageText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	var items []responsesMessageContent
	if err := json.Unmarshal(raw, &items); err == nil {
		return extractResponsesMessageText(items)
	}
	return ""
}

func extractResponsesMessageText(items []responsesMessageContent) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Text) == "" {
			continue
		}
		if item.Type == "output_text" || item.Type == "text" || item.Type == "" {
			parts = append(parts, strings.TrimSpace(item.Text))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func extractReasoningSummary(items []responsesReasoningSummary) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if text := strings.TrimSpace(item.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

type chatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []chatCompletionMessage `json:"messages"`
	Temperature float64                 `json:"temperature,omitempty"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content          json.RawMessage `json:"content"`
			ReasoningContent string          `json:"reasoning_content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type responsesRequest struct {
	Model     string               `json:"model"`
	Input     []responsesInputItem `json:"input"`
	Reasoning *responsesReasoning  `json:"reasoning,omitempty"`
}

type responsesReasoning struct {
	Summary string `json:"summary,omitempty"`
}

type responsesInputItem struct {
	Role    string                  `json:"role"`
	Content []responsesInputContent `json:"content"`
}

type responsesInputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesResponse struct {
	OutputText string                `json:"output_text"`
	Output     []responsesOutputItem `json:"output"`
	Usage      struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type responsesOutputItem struct {
	Type    string                      `json:"type"`
	Content []responsesMessageContent   `json:"content"`
	Summary []responsesReasoningSummary `json:"summary"`
}

type responsesMessageContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesReasoningSummary struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
