package ollama

import (
	"bufio"
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

const defaultBaseURL = "http://localhost:11434" // canonical default; also defined as config.DefaultOllamaEndpoint
const reconnectThreshold = time.Second
const defaultModel = "qwen2.5:3b" // canonical default; also defined as config.DefaultModel
const defaultRequestTimeout = 30 * time.Second
const availabilityTimeout = 3 * time.Second
const modelMetadataTimeout = 10 * time.Second
const defaultGenerateTimeout = 300 * time.Second // 5 minutes default for generation

// OllamaClient implements llm.LLMProvider for Ollama local models.
type OllamaClient struct {
	baseURL        string
	model          string
	primary        string
	secondary      string
	contextLength  int           // num_ctx to use; 0 = use defaultContextLength
	generateTimeout time.Duration // configurable timeout for generation requests
	httpClient     *http.Client
	lastOK         time.Time
	mu             sync.RWMutex
}

const defaultContextLength = 32768

// NewClient creates an OllamaClient. Uses http://localhost:11434 by default.
// contextLength of 0 means use defaultContextLength (32768).
func NewClient(baseURL, model string, contextLength ...int) *OllamaClient {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if model == "" {
		model = defaultModel
	}
	ctxLen := 0
	if len(contextLength) > 0 {
		ctxLen = contextLength[0]
	}
	return &OllamaClient{
		baseURL:         baseURL,
		model:           model,
		primary:         model,
		contextLength:   ctxLen,
		generateTimeout: defaultGenerateTimeout,
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: reconnectThreshold,
			},
		},
		lastOK: time.Time{},
	}
}

// SetGenerateTimeout sets a custom timeout for LLM generation requests.
func (c *OllamaClient) SetGenerateTimeout(d time.Duration) {
	c.mu.Lock()
	c.generateTimeout = d
	c.mu.Unlock()
}

func (c *OllamaClient) resolveGenerateTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.generateTimeout > 0 {
		return c.generateTimeout
	}
	return defaultGenerateTimeout
}

func (c *OllamaClient) Name() string {
	return "ollama"
}

// SetContextLength updates the context length used for requests.
func (c *OllamaClient) SetContextLength(n int) {
	c.mu.Lock()
	c.contextLength = n
	c.mu.Unlock()
}

func (c *OllamaClient) resolveContextLength() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.contextLength > 0 {
		return c.contextLength
	}
	return defaultContextLength
}

// DetectModelContext queries Ollama /api/show to extract the model's context
// length from its modelfile parameters. Returns 0 if not detectable.
func (c *OllamaClient) DetectModelContext(ctx context.Context, modelName string) int {
	if modelName == "" {
		modelName = c.model
	}
	ctx, cancel := withTimeoutIfMissing(ctx, modelMetadataTimeout)
	defer cancel()

	body := OllamaShowRequest{Name: modelName}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return 0
	}
	resp, err := c.doRequest(ctx, "POST", "/api/show", bytes.NewReader(jsonBody))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	var out OllamaShowResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0
	}
	return parseContextLengthFromModelfile(out.Parameters)
}

func parseContextLengthFromModelfile(params string) int {
	for _, line := range strings.Split(params, "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "num_ctx" {
			var n int
			if _, err := fmt.Sscanf(fields[1], "%d", &n); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

// IsAvailable checks if Ollama is reachable via GET /api/version.
func (c *OllamaClient) IsAvailable(ctx context.Context) bool {
	c.mu.Lock()
	if time.Since(c.lastOK) < reconnectThreshold {
		c.mu.Unlock()
		return true
	}
	c.mu.Unlock()

	ctx, cancel := withTimeoutIfMissing(ctx, availabilityTimeout)
	defer cancel()

	resp, err := c.doRequest(ctx, "GET", "/api/version", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	c.mu.Lock()
	c.lastOK = time.Now()
	c.mu.Unlock()
	return true
}

// Generate sends a prompt to POST /api/chat and returns the full response.
func (c *OllamaClient) Generate(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
	ctx, cancel := withTimeoutIfMissing(ctx, c.resolveGenerateTimeout())
	defer cancel()

	model := c.resolveModel(req)
	messages := []OllamaChatMessage{}
	if system := strings.TrimSpace(req.System); system != "" {
		messages = append(messages, OllamaChatMessage{Role: "system", Content: system})
	}
	messages = append(messages, OllamaChatMessage{Role: "user", Content: req.Prompt})
	body := OllamaChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
		Options: &OllamaGenerateOptions{
			NumGPU:      -1,
			Temperature: req.Temperature,
			NumCtx:      c.resolveContextLength(),
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}
	resp, err := c.doRequest(ctx, "POST", "/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		detail := strings.TrimSpace(string(errBody))
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("ollama: model %q not found (404). Run 'ollama pull %s' to download it. %s", model, model, detail)
		}
		return nil, fmt.Errorf("ollama: chat failed: status %d: %s", resp.StatusCode, detail)
	}
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama: read response: %w", err)
	}
	var out OllamaChatResponse
	if err := json.Unmarshal(rawBody, &out); err != nil {
		// Ollama may still return NDJSON despite stream:false — parse line by line.
		merged, mergeErr := mergeNDJSON(rawBody)
		if mergeErr != nil {
			return nil, fmt.Errorf("ollama: decode response: %w (raw length %d)", err, len(rawBody))
		}
		c.recordSuccess()
		return merged, nil
	}
	c.recordSuccess()
	return &llm.GenerateResponse{
		Text:       out.Message.Content,
		Thinking:   strings.TrimSpace(out.Message.Thinking),
		Raw:        string(rawBody),
		TokenCount: out.EvalCount,
	}, nil
}

// mergeNDJSON handles the case where Ollama returns line-delimited JSON
// (streaming format) despite stream:false. It concatenates all chunk texts.
func mergeNDJSON(data []byte) (*llm.GenerateResponse, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var text strings.Builder
	var thinking strings.Builder
	var evalCount int
	parsed := 0
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var chunk OllamaChatStreamChunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}
		parsed++
		text.WriteString(chunk.Message.Content)
		if chunk.Message.Thinking != "" {
			thinking.WriteString(chunk.Message.Thinking)
		}
		if chunk.Done {
			evalCount = chunk.EvalCount
		}
	}
	if parsed == 0 {
		return nil, fmt.Errorf("no valid JSON objects found")
	}
	return &llm.GenerateResponse{
		Text:       strings.TrimSpace(text.String()),
		Thinking:   strings.TrimSpace(thinking.String()),
		Raw:        string(data),
		TokenCount: evalCount,
	}, nil
}

// GenerateStream sends a prompt to POST /api/chat with stream=true and yields chunks via channel.
func (c *OllamaClient) GenerateStream(ctx context.Context, req llm.GenerateRequest) (<-chan llm.StreamChunk, error) {
	ctx, cancel := withTimeoutIfMissing(ctx, c.resolveGenerateTimeout())

	model := c.resolveModel(req)
	messages := []OllamaChatMessage{}
	if system := strings.TrimSpace(req.System); system != "" {
		messages = append(messages, OllamaChatMessage{Role: "system", Content: system})
	}
	messages = append(messages, OllamaChatMessage{Role: "user", Content: req.Prompt})
	body := OllamaChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
		Options: &OllamaGenerateOptions{
			NumGPU:      -1,
			Temperature: req.Temperature,
			NumCtx:      c.resolveContextLength(),
		},
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}
	resp, err := c.doRequest(ctx, "POST", "/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		detail := strings.TrimSpace(string(errBody))
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("ollama: model %q not found (404). Run 'ollama pull %s' to download it. %s", model, model, detail)
		}
		return nil, fmt.Errorf("ollama: chat stream failed: status %d: %s", resp.StatusCode, detail)
	}
	ch := make(chan llm.StreamChunk, 16)
	go func() {
		defer cancel()
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var chunk OllamaChatStreamChunk
			if err := json.Unmarshal(line, &chunk); err != nil {
				continue
			}
			select {
			case ch <- llm.StreamChunk{Text: chunk.Message.Content, Thinking: chunk.Message.Thinking, Done: chunk.Done}:
			case <-ctx.Done():
				return
			}
			if chunk.Done {
				c.recordSuccess()
				return
			}
		}
	}()
	return ch, nil
}

// ModelInfo returns model info via POST /api/show.
func (c *OllamaClient) ModelInfo(ctx context.Context) (*llm.ModelInfo, error) {
	ctx, cancel := withTimeoutIfMissing(ctx, modelMetadataTimeout)
	defer cancel()

	body := OllamaShowRequest{Name: c.model}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal show request: %w", err)
	}
	resp, err := c.doRequest(ctx, "POST", "/api/show", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("ollama: model %q not found. Run 'ollama pull %s'", c.model, c.model)
		}
		return nil, fmt.Errorf("ollama: show failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}
	var out OllamaShowResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ollama: decode show response: %w", err)
	}
	c.recordSuccess()
	return &llm.ModelInfo{
		Name:      c.model,
		Provider:  c.Name(),
		Size:      0,
		Family:    out.Details.Family,
		ParamSize: out.Details.ParameterSize,
		Quant:     out.Details.QuantizationLevel,
	}, nil
}

// ListModels returns all locally available models via GET /api/tags.
func (c *OllamaClient) ListModels(ctx context.Context) ([]llm.ModelInfo, error) {
	ctx, cancel := withTimeoutIfMissing(ctx, modelMetadataTimeout)
	defer cancel()

	resp, err := c.doRequest(ctx, "GET", "/api/tags", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: list models failed: status %d", resp.StatusCode)
	}
	var out OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ollama: decode tags response: %w", err)
	}
	c.recordSuccess()
	models := make([]llm.ModelInfo, 0, len(out.Models))
	for _, m := range out.Models {
		models = append(models, llm.ModelInfo{
			Name:      m.Name,
			Provider:  c.Name(),
			Size:      m.Size,
			Family:    m.Details.Family,
			ParamSize: m.Details.ParameterSize,
			Quant:     m.Details.QuantizationLevel,
		})
	}
	return models, nil
}

// SetModel switches the active model at runtime.
func (c *OllamaClient) SetModel(name string) {
	c.mu.Lock()
	c.model = name
	c.primary = name
	c.mu.Unlock()
}

func (c *OllamaClient) SetModelForRole(role llm.ModelRole, name string) {
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

func (c *OllamaClient) recordSuccess() {
	c.mu.Lock()
	c.lastOK = time.Now()
	c.mu.Unlock()
}

func (c *OllamaClient) resolveModel(req llm.GenerateRequest) string {
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
		if strings.TrimSpace(c.primary) != "" {
			return c.primary
		}
	default:
		if strings.TrimSpace(c.primary) != "" {
			return c.primary
		}
	}
	if strings.TrimSpace(c.model) != "" {
		return c.model
	}
	return defaultModel
}

func (c *OllamaClient) doRequest(ctx context.Context, method, path string, body *bytes.Reader) (*http.Response, error) {
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return nil, fmt.Errorf("ollama: invalid url: %w", err)
	}
	var reqBody io.Reader
	if body != nil {
		reqBody = body
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama: new request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: request failed: %w", err)
	}
	return resp, nil
}

func withTimeoutIfMissing(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	if d <= 0 {
		d = defaultRequestTimeout
	}
	return context.WithTimeout(ctx, d)
}
