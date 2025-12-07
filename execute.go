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
	"sync"
	"time"
)

const (
	maxRetries = 3
	baseDelay  = time.Second
)

// Execute fires parallel requests to all configured providers.
// Results are streamed into the returned channel as each provider responds.
// The channel is closed when all providers have responded.
func (c *Command) Execute(req ChatCompletionRequest) <-chan Result {
	results := make(chan Result, len(c.providers))

	c.log(slog.LevelDebug, "starting parallel requests",
		"providers", len(c.providers),
	)

	var wg sync.WaitGroup
	for _, provider := range c.providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			c.executeAndSend(p, req, results)
		}(provider)
	}

	go func() {
		wg.Wait()
		close(results)
		c.log(slog.LevelDebug, "all providers completed")
	}()

	return results
}

// ExecuteOne sends a request to the first configured provider and blocks until complete.
// Useful for simple cases and debugging.
func (c *Command) ExecuteOne(req ChatCompletionRequest) (ChatCompletionResponse, error) {
	if len(c.providers) == 0 {
		return ChatCompletionResponse{}, fmt.Errorf("no providers configured")
	}
	return c.executeProvider(c.providers[0], req)
}

// executeProvider sends a request to a specific provider.
func (c *Command) executeProvider(provider Provider, req ChatCompletionRequest) (ChatCompletionResponse, error) {
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

	return c.executeWithRetry(provider, requestBody)
}

func (c *Command) executeAndSend(provider Provider, req ChatCompletionRequest, results chan<- Result) {
	start := time.Now()

	resp, err := c.executeProvider(provider, req)
	duration := time.Since(start)

	result := Result{
		Provider: provider.Name,
		Response: resp,
		Error:    err,
		Duration: duration,
	}

	results <- result

	if err != nil {
		c.log(slog.LevelWarn, "provider failed",
			"provider", provider.Name,
			"duration", duration,
			"error", err.Error(),
		)
	} else {
		c.log(slog.LevelDebug, "provider responded",
			"provider", provider.Name,
			"duration", duration,
		)
	}
}

func (c *Command) executeWithRetry(provider Provider, requestBody []byte) (ChatCompletionResponse, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := c.executeSingleRequest(provider, requestBody)
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

		time.Sleep(time.Duration(1<<uint(attempt)) * baseDelay)
	}

	return ChatCompletionResponse{}, fmt.Errorf("request to %s failed after %d attempts: %w", provider.Name, maxRetries, lastErr)
}

func (c *Command) executeSingleRequest(provider Provider, requestBody []byte) (ChatCompletionResponse, error) {
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", provider.Endpoint, bytes.NewBuffer(requestBody))
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

	if strings.Contains(errStr, "HTTP request failed") {
		return true
	}

	if strings.Contains(errStr, "API request failed with status") {
		if strings.Contains(errStr, "status 5") {
			return true
		}
		return false
	}

	if strings.Contains(errStr, "failed to decode response") {
		return true
	}

	return false
}
