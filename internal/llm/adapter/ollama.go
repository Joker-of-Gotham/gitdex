package adapter

const (
	OllamaDefaultEndpoint = "http://localhost:11434/v1"
	OllamaDefaultModel    = "llama3.2"
)

func NewOllamaProvider(cfg OpenAIConfig) *OpenAIProvider {
	if cfg.Endpoint == "" {
		cfg.Endpoint = OllamaDefaultEndpoint
	}
	if cfg.Model == "" {
		cfg.Model = OllamaDefaultModel
	}
	return NewOpenAIProvider(cfg)
}
