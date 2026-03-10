package ollama

// Ollama API request/response types.

// OllamaVersionResponse is the response from GET /api/version
type OllamaVersionResponse struct {
	Version string `json:"version"`
}

// OllamaGenerateOptions controls runtime inference parameters.
type OllamaGenerateOptions struct {
	NumGPU      int     `json:"num_gpu,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	NumCtx      int     `json:"num_ctx,omitempty"`
}

// OllamaGenerateRequest is the request body for POST /api/generate
type OllamaGenerateRequest struct {
	Model   string                 `json:"model"`
	System  string                 `json:"system,omitempty"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream,omitempty"`
	Options *OllamaGenerateOptions `json:"options,omitempty"`
}

// OllamaGenerateResponse is the non-streaming response from POST /api/generate
type OllamaGenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at,omitempty"`
	EvalCount int    `json:"eval_count,omitempty"`
}

// OllamaGenerateStreamChunk is a chunk from streaming POST /api/generate
type OllamaGenerateStreamChunk struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at,omitempty"`
	EvalCount int    `json:"eval_count,omitempty"`
}

// OllamaShowRequest is the request body for POST /api/show
type OllamaShowRequest struct {
	Name string `json:"name"`
}

// OllamaShowResponse is the response from POST /api/show
type OllamaShowResponse struct {
	Modelfile  string `json:"modelfile,omitempty"`
	Parameters string `json:"parameters,omitempty"`
	Details    struct {
		Format            string `json:"format,omitempty"`
		Family            string `json:"family,omitempty"`
		ParameterSize     string `json:"parameter_size,omitempty"`
		QuantizationLevel string `json:"quantization_level,omitempty"`
	} `json:"details,omitempty"`
}

// OllamaTagsResponse is the response from GET /api/tags
type OllamaTagsResponse struct {
	Models []OllamaModelEntry `json:"models"`
}

// OllamaModelEntry represents a single model from the tags API.
type OllamaModelEntry struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Details    struct {
		Format            string `json:"format,omitempty"`
		Family            string `json:"family,omitempty"`
		ParameterSize     string `json:"parameter_size,omitempty"`
		QuantizationLevel string `json:"quantization_level,omitempty"`
	} `json:"details"`
}
