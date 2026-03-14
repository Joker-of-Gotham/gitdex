package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientGenerate_DeepSeekReasoningContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/chat/completions", r.URL.Path)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"analysis\":\"ok\",\"suggestions\":[]}","reasoning_content":"inspect repo, compare remotes"}}],"usage":{"total_tokens":123}}`)
	}))
	defer server.Close()

	client := NewClient(ProviderDeepSeek, server.URL, "secret", "deepseek-reasoner", 5*time.Second)
	resp, err := client.Generate(context.Background(), llm.GenerateRequest{
		Model:  "deepseek-reasoner",
		System: "sys",
		Prompt: "user",
	})
	require.NoError(t, err)
	assert.Equal(t, `{"analysis":"ok","suggestions":[]}`, resp.Text)
	assert.Equal(t, "inspect repo, compare remotes", resp.Thinking)
	assert.Contains(t, resp.Raw, "reasoning_content")
}

func TestClientGenerate_OpenAIResponsesSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/responses", r.URL.Path)
		fmt.Fprint(w, `{"output_text":"{\"analysis\":\"ok\",\"suggestions\":[]}","output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"Checked branch state and remote health."}]},{"type":"message","content":[{"type":"output_text","text":"{\"analysis\":\"ok\",\"suggestions\":[]}"}]}],"usage":{"total_tokens":88}}`)
	}))
	defer server.Close()

	client := NewClient(ProviderOpenAI, server.URL, "secret", "o4-mini", 5*time.Second)
	resp, err := client.Generate(context.Background(), llm.GenerateRequest{
		Model:  "o4-mini",
		System: "sys",
		Prompt: "user",
	})
	require.NoError(t, err)
	assert.Equal(t, `{"analysis":"ok","suggestions":[]}`, resp.Text)
	assert.Equal(t, "Checked branch state and remote health.", resp.Thinking)
	assert.Contains(t, resp.Raw, "\"summary\"")
}
