package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/llm"
)

func TestParseContextLengthFromModelfile(t *testing.T) {
	if got := parseContextLengthFromModelfile("temperature 0.1\nnum_ctx 16384\n"); got != 16384 {
		t.Fatalf("unexpected context length: %d", got)
	}
}

func TestResolveModelPrefersSecondary(t *testing.T) {
	client := NewClient("http://localhost:11434", "primary")
	client.SetModelForRole(llm.RoleSecondary, "secondary")
	if got := client.resolveModel(llm.GenerateRequest{Role: llm.RoleSecondary}); got != "secondary" {
		t.Fatalf("unexpected model: %s", got)
	}
}

func TestStreamFieldNotOmitted(t *testing.T) {
	req := OllamaChatRequest{
		Model:    "test",
		Messages: []OllamaChatMessage{{Role: "user", Content: "hi"}},
		Stream:   false,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"stream":false`) {
		t.Errorf("expected stream:false in JSON, got: %s", string(data))
	}
}

func TestStreamFieldTrue(t *testing.T) {
	req := OllamaChatRequest{
		Model:    "test",
		Messages: []OllamaChatMessage{{Role: "user", Content: "hi"}},
		Stream:   true,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"stream":true`) {
		t.Errorf("expected stream:true in JSON, got: %s", string(data))
	}
}

func TestMergeNDJSON_SingleObject(t *testing.T) {
	obj := `{"model":"test","message":{"role":"assistant","content":"Hello world"},"done":true,"eval_count":5}`
	resp, err := mergeNDJSON([]byte(obj))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", resp.Text)
	}
	if resp.TokenCount != 5 {
		t.Errorf("expected token count 5, got %d", resp.TokenCount)
	}
}

func TestMergeNDJSON_MultipleChunks(t *testing.T) {
	lines := strings.Join([]string{
		`{"model":"test","message":{"role":"assistant","content":"Hello "},"done":false}`,
		`{"model":"test","message":{"role":"assistant","content":"world"},"done":true,"eval_count":10}`,
	}, "\n")
	resp, err := mergeNDJSON([]byte(lines))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", resp.Text)
	}
	if resp.TokenCount != 10 {
		t.Errorf("expected token count 10, got %d", resp.TokenCount)
	}
}

func TestMergeNDJSON_WithThinking(t *testing.T) {
	lines := strings.Join([]string{
		`{"model":"test","message":{"role":"assistant","content":"","thinking":"Let me think..."},"done":false}`,
		`{"model":"test","message":{"role":"assistant","content":"Answer"},"done":true,"eval_count":3}`,
	}, "\n")
	resp, err := mergeNDJSON([]byte(lines))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Answer" {
		t.Errorf("expected 'Answer', got %q", resp.Text)
	}
	if resp.Thinking != "Let me think..." {
		t.Errorf("expected thinking 'Let me think...', got %q", resp.Thinking)
	}
}

func TestMergeNDJSON_EmptyInput(t *testing.T) {
	_, err := mergeNDJSON([]byte(""))
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestMergeNDJSON_InvalidJSON(t *testing.T) {
	_, err := mergeNDJSON([]byte("not json at all"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGenerate_SingleResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Stream {
			t.Error("expected stream=false in request")
		}
		resp := OllamaChatResponse{
			Model:   "test",
			Message: OllamaChatMessage{Role: "assistant", Content: "pong"},
			Done:    true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test")
	resp, err := client.Generate(context.Background(), llm.GenerateRequest{
		System: "test",
		Prompt: "ping",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "pong" {
		t.Errorf("expected 'pong', got %q", resp.Text)
	}
}

func TestGenerate_NDJSONFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate Ollama ignoring stream:false and returning NDJSON
		fmt.Fprintln(w, `{"model":"test","message":{"role":"assistant","content":"Hello "},"done":false}`)
		fmt.Fprintln(w, `{"model":"test","message":{"role":"assistant","content":"world"},"done":true,"eval_count":7}`)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test")
	resp, err := client.Generate(context.Background(), llm.GenerateRequest{
		System: "test",
		Prompt: "ping",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", resp.Text)
	}
	if resp.TokenCount != 7 {
		t.Errorf("expected token count 7, got %d", resp.TokenCount)
	}
}

func TestGenerate_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test")
	_, err := client.Generate(context.Background(), llm.GenerateRequest{Prompt: "ping"})
	if err == nil {
		t.Error("expected error for 500 status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code in error, got: %s", err.Error())
	}
}

func TestGenerate_404_ModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OllamaChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"error":"model '%s' not found, try pulling it first"}`, req.Model)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "nonexistent-model:7b")
	_, err := client.Generate(context.Background(), llm.GenerateRequest{Prompt: "ping"})
	if err == nil {
		t.Fatal("expected error for 404")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "not found") {
		t.Errorf("expected 'not found' in error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "ollama pull") {
		t.Errorf("expected 'ollama pull' hint in error, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "nonexistent-model:7b") {
		t.Errorf("expected model name in error, got: %s", errMsg)
	}
}

func TestGenerateStream_404_ModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "missing:latest")
	_, err := client.GenerateStream(context.Background(), llm.GenerateRequest{Prompt: "ping"})
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err.Error())
	}
}

func TestListModels_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(OllamaTagsResponse{
			Models: []OllamaModelEntry{
				{Name: "qwen2.5:3b", Size: 1000, Details: struct {
					Format            string `json:"format,omitempty"`
					Family            string `json:"family,omitempty"`
					ParameterSize     string `json:"parameter_size,omitempty"`
					QuantizationLevel string `json:"quantization_level,omitempty"`
				}{Family: "qwen", ParameterSize: "3B", QuantizationLevel: "Q4_K_M"}},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test")
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].Name != "qwen2.5:3b" {
		t.Errorf("expected 'qwen2.5:3b', got %q", models[0].Name)
	}
	if models[0].Family != "qwen" {
		t.Errorf("expected family 'qwen', got %q", models[0].Family)
	}
}

func TestIsAvailable_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(OllamaVersionResponse{Version: "0.1.0"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test")
	if !client.IsAvailable(context.Background()) {
		t.Error("expected available")
	}
}

func TestIsAvailable_Fail(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", "test")
	if client.IsAvailable(context.Background()) {
		t.Error("expected not available")
	}
}
