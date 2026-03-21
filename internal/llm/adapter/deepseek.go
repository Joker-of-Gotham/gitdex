package adapter

const (
	DeepSeekDefaultEndpoint = "https://api.deepseek.com/v1"
	DeepSeekDefaultModel    = "deepseek-chat"
)

func NewDeepSeekProvider(cfg OpenAIConfig) *OpenAIProvider {
	if cfg.Endpoint == "" {
		cfg.Endpoint = DeepSeekDefaultEndpoint
	}
	if cfg.Model == "" {
		cfg.Model = DeepSeekDefaultModel
	}
	return NewOpenAIProvider(cfg)
}
