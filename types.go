package general

import (
	"encoding/json"
	"strings"
	"time"
)

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
	Role       string         `json:"role"`
	Content    MessageContent `json:"content,omitempty"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

// MessageContent holds either plain text or multipart content (text + images).
// Serializes to a JSON string when text-only, or a JSON array when multipart.
type MessageContent struct {
	Text  string        // used when content is a simple string
	Parts []ContentPart // used when content is multipart (text + images)
}

// TextContent creates a text-only MessageContent.
func TextContent(text string) MessageContent {
	return MessageContent{Text: text}
}

// MultiContent creates a multipart MessageContent from parts.
func MultiContent(parts ...ContentPart) MessageContent {
	return MessageContent{Parts: parts}
}

// String returns the text content. For multipart, concatenates all text parts.
func (c MessageContent) String() string {
	if c.Parts == nil {
		return c.Text
	}
	var sb strings.Builder
	for _, p := range c.Parts {
		if p.Type == "text" {
			sb.WriteString(p.Text)
		}
	}
	return sb.String()
}

// IsZero returns true if the content is empty.
func (c MessageContent) IsZero() bool {
	return c.Text == "" && len(c.Parts) == 0
}

func (c MessageContent) MarshalJSON() ([]byte, error) {
	if c.Parts != nil {
		return json.Marshal(c.Parts)
	}
	return json.Marshal(c.Text)
}

func (c *MessageContent) UnmarshalJSON(data []byte) error {
	// Try string first (most common case from API responses)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.Text = s
		c.Parts = nil
		return nil
	}

	// Try array of content parts
	var parts []ContentPart
	if err := json.Unmarshal(data, &parts); err == nil {
		c.Parts = parts
		c.Text = ""
		return nil
	}

	return nil // empty content
}

// ContentPart represents one part of a multimodal message.
type ContentPart struct {
	Type     string    `json:"type"`               // "text" or "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL holds an image reference, typically a data URI.
type ImageURL struct {
	URL string `json:"url"`
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

// Provider represents an LLM API endpoint.
type Provider struct {
	Endpoint string
	APIKey   string
	Headers  map[string]string
}

// Target is a specific provider + model combination.
type Target struct {
	Provider Provider
	Model    string
}

// Result wraps a response with target info and timing.
type Result struct {
	Target   Target
	Response ChatCompletionResponse
	Error    error
	Duration time.Duration
}
