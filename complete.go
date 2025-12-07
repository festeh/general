package general

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	maxRetries = 3
	baseDelay  = time.Second
)

// Execute sends a chat completion request to a single provider.
// Returns the response or an error after retries are exhausted.
func (c *Command) Execute(ctx context.Context, provider Provider, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	// Override model with provider's model
	req.Model = provider.Model

	requestBody, err := json.Marshal(req)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.log(slog.LevelDebug, "sending request",
		"provider", provider.Name,
		"endpoint", provider.Endpoint,
		"model", provider.Model,
	)

	return c.executeWithRetry(ctx, provider, requestBody)
}

func (c *Command) executeWithRetry(ctx context.Context, provider Provider, requestBody []byte) (ChatCompletionResponse, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := c.executeSingleRequest(ctx, provider, requestBody)
		if err == nil {
			return result, nil
		}

		lastErr = err
		c.log(slog.LevelWarn, "request attempt failed",
			"provider", provider.Name,
			"attempt", attempt+1,
			"error", err.Error(),
		)

		if attempt == maxRetries-1 {
			break
		}

		if !shouldRetry(err) {
			break
		}

		if err := c.waitForRetry(ctx, attempt); err != nil {
			return ChatCompletionResponse{}, err
		}
	}

	return ChatCompletionResponse{}, fmt.Errorf("request to %s failed after %d attempts: %w", provider.Name, maxRetries, lastErr)
}

func (c *Command) executeSingleRequest(ctx context.Context, provider Provider, requestBody []byte) (ChatCompletionResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		var responseBody []byte
		if httpResp.Body != nil {
			responseBody, _ = io.ReadAll(httpResp.Body)
		}
		return ChatCompletionResponse{}, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode, string(responseBody))
	}

	var response ChatCompletionResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&response); err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return ChatCompletionResponse{}, fmt.Errorf("no choices in response")
	}

	c.log(slog.LevelDebug, "request successful",
		"provider", provider.Name,
		"choices", len(response.Choices),
	)

	return response, nil
}

func shouldRetry(err error) bool {
	errStr := err.Error()

	// Retry on network errors
	if strings.Contains(errStr, "HTTP request failed") {
		return true
	}

	// Retry on server errors (5xx), but not client errors (4xx)
	if strings.Contains(errStr, "API request failed with status") {
		if strings.Contains(errStr, "status 5") {
			return true
		}
		return false
	}

	// Retry on decode errors (could be temporary network issues)
	if strings.Contains(errStr, "failed to decode response") {
		return true
	}

	return false
}

func (c *Command) waitForRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(1<<uint(attempt)) * baseDelay

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}
