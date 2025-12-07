package general

import "time"

// ChatCompletionRequest represents an OpenAI-compatible chat completion request.
type ChatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []ChatCompletionMessage `json:"messages"`
	MaxTokens   int                     `json:"max_tokens,omitempty"`
	Temperature float64                 `json:"temperature,omitempty"`
	Tools       []Tool                  `json:"tools,omitempty"`
	ToolChoice  any                     `json:"tool_choice,omitempty"`
}

// ChatCompletionMessage represents a message in the conversation.
type ChatCompletionMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ChatCompletionResponse represents an OpenAI-compatible chat completion response.
type ChatCompletionResponse struct {
	Choices []ChatCompletionChoice `json:"choices"`
}

// ChatCompletionChoice represents a single choice in the response.
type ChatCompletionChoice struct {
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason,omitempty"`
}

// ToolCall represents a tool call made by the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction contains the function name and arguments for a tool call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool represents a tool definition for the model.
type Tool struct {
	Type     string   `json:"type"`
	Function ToolFunc `json:"function"`
}

// ToolFunc describes a function that can be called by the model.
type ToolFunc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
}

// ToolParameters describes the parameters for a tool function.
type ToolParameters struct {
	Type       string                           `json:"type"`
	Properties map[string]ToolParameterProperty `json:"properties"`
	Required   []string                         `json:"required,omitempty"`
}

// ToolParameterProperty describes a single parameter property.
type ToolParameterProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// Provider represents an LLM endpoint configuration.
type Provider struct {
	Name     string
	Endpoint string
	APIKey   string
	Model    string
}

// Result wraps a response with provider info and timing.
type Result struct {
	Provider string
	Response ChatCompletionResponse
	Error    error
	Duration time.Duration
}
