package general

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Broadcast fires parallel requests to all configured providers.
// Results are streamed into the returned channel as each provider responds.
// The channel is closed when all providers have responded.
func (c *Command) Broadcast(ctx context.Context, req ChatCompletionRequest) <-chan Result {
	results := make(chan Result, len(c.providers))

	c.log(slog.LevelDebug, "starting parallel requests",
		"providers", len(c.providers),
	)

	var wg sync.WaitGroup
	for _, provider := range c.providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			c.executeAndSend(ctx, p, req, results)
		}(provider)
	}

	go func() {
		wg.Wait()
		close(results)
		c.log(slog.LevelDebug, "all providers completed")
	}()

	return results
}

func (c *Command) executeAndSend(ctx context.Context, provider Provider, req ChatCompletionRequest, results chan<- Result) {
	start := time.Now()

	resp, err := c.Execute(ctx, provider, req)
	duration := time.Since(start)

	result := Result{
		Provider: provider.Name,
		Response: resp,
		Error:    err,
		Duration: duration,
	}

	select {
	case results <- result:
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
	case <-ctx.Done():
		c.log(slog.LevelDebug, "context cancelled, dropping result",
			"provider", provider.Name,
		)
	}
}
