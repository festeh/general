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

// Execute fires parallel requests to all configured targets.
// Results are streamed into the returned channel as each target responds.
// The channel is closed when all targets have responded.
func (c *Command) Execute(req ChatCompletionRequest) <-chan Result {
	results := make(chan Result, len(c.targets))

	c.log(slog.LevelDebug, "starting parallel requests",
		"targets", len(c.targets),
	)

	var wg sync.WaitGroup
	for _, target := range c.targets {
		wg.Add(1)
		go func(t Target) {
			defer wg.Done()
			c.executeAndSend(t, req, results)
		}(target)
	}

	go func() {
		wg.Wait()
		close(results)
		c.log(slog.LevelDebug, "all targets completed")
	}()

	return results
}

// ExecuteOne sends a request to the first configured target and blocks until complete.
// Useful for simple cases and debugging.
func (c *Command) ExecuteOne(req ChatCompletionRequest) (ChatCompletionResponse, error) {
	if len(c.targets) == 0 {
		return ChatCompletionResponse{}, fmt.Errorf("no targets configured")
	}
	return c.executeTarget(c.targets[0], req)
}

// executeTarget sends a request to a specific target.
func (c *Command) executeTarget(target Target, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	req.Model = target.Model

	requestBody, err := json.Marshal(req)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.log(slog.LevelDebug, "sending request",
		"endpoint", target.Provider.Endpoint,
		"model", target.Model,
	)

	return c.executeWithRetry(target, requestBody)
}

func (c *Command) executeAndSend(target Target, req ChatCompletionRequest, results chan<- Result) {
	start := time.Now()

	resp, err := c.executeTarget(target, req)
	duration := time.Since(start)

	result := Result{
		Target:   target,
		Response: resp,
		Error:    err,
		Duration: duration,
	}

	results <- result

	if err != nil {
		c.log(slog.LevelWarn, "target failed",
			"endpoint", target.Provider.Endpoint,
			"model", target.Model,
			"duration", duration,
			"error", err.Error(),
		)
	} else {
		c.log(slog.LevelDebug, "target responded",
			"endpoint", target.Provider.Endpoint,
			"model", target.Model,
			"duration", duration,
		)
	}
}

func (c *Command) executeWithRetry(target Target, requestBody []byte) (ChatCompletionResponse, error) {
	var lastErr error

	for attempt := range maxRetries {
		result, err := c.executeSingleRequest(target, requestBody)
		if err == nil {
			return result, nil
		}

		lastErr = err
		c.log(slog.LevelWarn, "request attempt failed",
			"endpoint", target.Provider.Endpoint,
			"model", target.Model,
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

	return ChatCompletionResponse{}, fmt.Errorf("request to %s/%s failed after %d attempts: %w", target.Provider.Endpoint, target.Model, maxRetries, lastErr)
}

func (c *Command) executeSingleRequest(target Target, requestBody []byte) (ChatCompletionResponse, error) {
	httpReq, err := http.NewRequestWithContext(context.Background(), "POST", target.Provider.Endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+target.Provider.APIKey)

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
		"endpoint", target.Provider.Endpoint,
		"model", target.Model,
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
