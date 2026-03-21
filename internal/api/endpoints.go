package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const APIVersion = "v1"

// APIRequest represents a generic API request.
type APIRequest struct {
	RequestID  string          `json:"request_id" yaml:"request_id"`
	Endpoint   string          `json:"endpoint" yaml:"endpoint"`
	Method     string          `json:"method" yaml:"method"`
	Payload    json.RawMessage `json:"payload,omitempty" yaml:"payload,omitempty"`
	APIVersion string          `json:"api_version" yaml:"api_version"`
	Timestamp  time.Time       `json:"timestamp" yaml:"timestamp"`
}

// APIResponse represents a generic API response.
type APIResponse struct {
	RequestID  string          `json:"request_id" yaml:"request_id"`
	StatusCode int             `json:"status_code" yaml:"status_code"`
	Payload    json.RawMessage `json:"payload,omitempty" yaml:"payload,omitempty"`
	Errors     []APIError      `json:"errors,omitempty" yaml:"errors,omitempty"`
	Timestamp  time.Time       `json:"timestamp" yaml:"timestamp"`
}

// APIError represents an API error.
type APIError struct {
	Code    string `json:"code" yaml:"code"`
	Message string `json:"message" yaml:"message"`
	Field   string `json:"field,omitempty" yaml:"field,omitempty"`
}

// Endpoint represents a registered API endpoint.
type Endpoint struct {
	Path        string      `json:"path" yaml:"path"`
	Method      string      `json:"method" yaml:"method"`
	Handler     HandlerFunc `json:"-" yaml:"-"`
	Description string      `json:"description" yaml:"description"`
	APIVersion  string      `json:"api_version" yaml:"api_version"`
}

// HandlerFunc is the handler for an endpoint.
type HandlerFunc func(*APIRequest) (*APIResponse, error)

// APIRouter routes requests to handlers.
type APIRouter interface {
	Register(endpoint Endpoint)
	Handle(request *APIRequest) (*APIResponse, error)
	ListEndpoints() []Endpoint
}

// MemoryAPIRouter implements APIRouter using in-memory storage.
type MemoryAPIRouter struct {
	mu        sync.RWMutex
	endpoints map[string]Endpoint // key: "METHOD /path"
}

// NewMemoryAPIRouter creates a new MemoryAPIRouter with standard endpoints registered.
func NewMemoryAPIRouter() *MemoryAPIRouter {
	r := &MemoryAPIRouter{
		endpoints: make(map[string]Endpoint),
	}
	r.registerDefaults()
	return r
}

func (r *MemoryAPIRouter) key(method, path string) string {
	return method + " " + path
}

func (r *MemoryAPIRouter) Register(e Endpoint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(e.Method, e.Path)
	r.endpoints[k] = e
}

func (r *MemoryAPIRouter) Handle(request *APIRequest) (*APIResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	path := normalizePath(request.Endpoint)
	method := strings.ToUpper(strings.TrimSpace(request.Method))
	if method == "" {
		method = "POST"
	}

	r.mu.RLock()
	ep, ok := r.endpoints[r.key(method, path)]
	r.mu.RUnlock()

	if !ok || ep.Handler == nil {
		return &APIResponse{
			RequestID:  request.RequestID,
			StatusCode: 404,
			Errors:     []APIError{{Code: "not_found", Message: "endpoint not found: " + method + " " + path}},
			Timestamp:  time.Now().UTC(),
		}, nil
	}

	resp, err := ep.Handler(request)
	if err != nil {
		return &APIResponse{
			RequestID:  request.RequestID,
			StatusCode: 500,
			Errors:     []APIError{{Code: "handler_error", Message: err.Error()}},
			Timestamp:  time.Now().UTC(),
		}, nil
	}
	if resp == nil {
		return &APIResponse{
			RequestID:  request.RequestID,
			StatusCode: 500,
			Errors:     []APIError{{Code: "handler_error", Message: "handler returned nil response"}},
			Timestamp:  time.Now().UTC(),
		}, nil
	}
	if resp.RequestID == "" {
		resp.RequestID = request.RequestID
	}
	return resp, nil
}

func (r *MemoryAPIRouter) ListEndpoints() []Endpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Endpoint
	for _, ep := range r.endpoints {
		out = append(out, Endpoint{
			Path:        ep.Path,
			Method:      ep.Method,
			Description: ep.Description,
			APIVersion:  ep.APIVersion,
		})
	}
	return out
}

// Query is not supported on MemoryAPIRouter; use ProviderQueryRouter with a configured storage provider.
func (r *MemoryAPIRouter) Query(_ *QueryRequest) (*QueryResult, error) {
	return nil, fmt.Errorf("query requires a configured storage provider; use ProviderQueryRouter with storage.StorageProvider")
}

// GetResource is not supported on MemoryAPIRouter; use ProviderQueryRouter with a configured storage provider.
func (r *MemoryAPIRouter) GetResource(_ string, _ string) (*APIResponse, error) {
	return nil, fmt.Errorf("get resource requires a configured storage provider; use ProviderQueryRouter with storage.StorageProvider")
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/api/v1/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasPrefix(p, "/api/") {
		p = "/api/v1/" + strings.TrimPrefix(p, "/")
	}
	// Normalize :id to :id for path params
	return p
}

func (r *MemoryAPIRouter) registerDefaults() {
	r.Register(Endpoint{
		Path:        "/api/v1/intents",
		Method:      "POST",
		Handler:     handleSubmitIntent,
		Description: "Submit structured intent",
		APIVersion:  APIVersion,
	})
	r.Register(Endpoint{
		Path:        "/api/v1/plans",
		Method:      "POST",
		Handler:     handleSubmitPlan,
		Description: "Submit structured plan",
		APIVersion:  APIVersion,
	})
	r.Register(Endpoint{
		Path:        "/api/v1/tasks",
		Method:      "POST",
		Handler:     handleSubmitTask,
		Description: "Submit structured task",
		APIVersion:  APIVersion,
	})
}

func handleSubmitIntent(req *APIRequest) (*APIResponse, error) {
	id := uuid.New().String()
	if len(req.Payload) == 0 {
		return &APIResponse{
			RequestID:  req.RequestID,
			StatusCode: 400,
			Errors:     []APIError{{Code: "invalid_payload", Message: "payload is required", Field: "payload"}},
			Timestamp:  time.Now().UTC(),
		}, nil
	}
	payload := map[string]any{
		"id":         id,
		"intent":     req.Payload,
		"accepted":   true,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return &APIResponse{
			RequestID:  req.RequestID,
			StatusCode: 500,
			Errors:     []APIError{{Code: "serialization_error", Message: err.Error()}},
			Timestamp:  time.Now().UTC(),
		}, nil
	}
	return &APIResponse{
		RequestID:  req.RequestID,
		StatusCode: 201,
		Payload:    raw,
		Timestamp:  time.Now().UTC(),
	}, nil
}

func handleSubmitPlan(req *APIRequest) (*APIResponse, error) {
	id := uuid.New().String()
	if len(req.Payload) == 0 {
		return &APIResponse{
			RequestID:  req.RequestID,
			StatusCode: 400,
			Errors:     []APIError{{Code: "invalid_payload", Message: "payload is required", Field: "payload"}},
			Timestamp:  time.Now().UTC(),
		}, nil
	}
	payload := map[string]any{
		"id":         id,
		"plan":       req.Payload,
		"accepted":   true,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return &APIResponse{
			RequestID:  req.RequestID,
			StatusCode: 500,
			Errors:     []APIError{{Code: "serialization_error", Message: err.Error()}},
			Timestamp:  time.Now().UTC(),
		}, nil
	}
	return &APIResponse{
		RequestID:  req.RequestID,
		StatusCode: 201,
		Payload:    raw,
		Timestamp:  time.Now().UTC(),
	}, nil
}

func handleSubmitTask(req *APIRequest) (*APIResponse, error) {
	id := uuid.New().String()
	if len(req.Payload) == 0 {
		return &APIResponse{
			RequestID:  req.RequestID,
			StatusCode: 400,
			Errors:     []APIError{{Code: "invalid_payload", Message: "payload is required", Field: "payload"}},
			Timestamp:  time.Now().UTC(),
		}, nil
	}
	payload := map[string]any{
		"id":         id,
		"task":       req.Payload,
		"accepted":   true,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return &APIResponse{
			RequestID:  req.RequestID,
			StatusCode: 500,
			Errors:     []APIError{{Code: "serialization_error", Message: err.Error()}},
			Timestamp:  time.Now().UTC(),
		}, nil
	}
	return &APIResponse{
		RequestID:  req.RequestID,
		StatusCode: 201,
		Payload:    raw,
		Timestamp:  time.Now().UTC(),
	}, nil
}
