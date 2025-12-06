package general

import (
	"log/slog"
	"net/http"
	"time"
)

const defaultTimeout = 60 * time.Second

// Client manages LLM API requests.
type Client struct {
	providers []Provider
	client    *http.Client
	logger    *slog.Logger
}

// NewClient creates a new Client with the given providers and optional logger.
// Pass nil for logger to disable logging.
func NewClient(providers []Provider, logger *slog.Logger) *Client {
	return NewClientWithTimeout(providers, logger, defaultTimeout)
}

// NewClientWithTimeout creates a new Client with a custom timeout.
func NewClientWithTimeout(providers []Provider, logger *slog.Logger, timeout time.Duration) *Client {
	return &Client{
		providers: providers,
		client:    &http.Client{Timeout: timeout},
		logger:    logger,
	}
}

// log logs a message if logger is configured.
func (c *Client) log(level slog.Level, msg string, args ...any) {
	if c.logger != nil {
		c.logger.Log(nil, level, msg, args...)
	}
}
