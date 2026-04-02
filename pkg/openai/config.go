package openai

// OpenAIConfig holds runtime configuration for OpenAI-compatible endpoints.
type OpenAIConfig struct {
	ModelID string `json:"model_id"`
	URL     string `json:"url"`
	APIKey  string `json:"api_key"`
	SSE     bool   `json:"sse"`
}
