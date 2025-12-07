package general

// Pre-configured endpoints for popular providers
const (
	OpenRouterEndpoint = "https://openrouter.ai/api/v1/chat/completions"
	GroqEndpoint       = "https://api.groq.com/openai/v1/chat/completions"
	ChutesEndpoint     = "https://llm.chutes.ai/v1/chat/completions"
	GeminiEndpoint     = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
)

// OpenRouter returns a Provider for OpenRouter API.
func OpenRouter(apiKey string) Provider {
	return Provider{Endpoint: OpenRouterEndpoint, APIKey: apiKey}
}

// Groq returns a Provider for Groq API.
func Groq(apiKey string) Provider {
	return Provider{Endpoint: GroqEndpoint, APIKey: apiKey}
}

// Chutes returns a Provider for Chutes AI API.
func Chutes(apiKey string) Provider {
	return Provider{Endpoint: ChutesEndpoint, APIKey: apiKey}
}

// Gemini returns a Provider for Google Gemini API (OpenAI-compatible mode).
func Gemini(apiKey string) Provider {
	return Provider{Endpoint: GeminiEndpoint, APIKey: apiKey}
}
