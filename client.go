package general

import (
	"log/slog"
	"net/http"
	"time"
)

const defaultTimeout = 60 * time.Second

// Command manages LLM API requests.
type Command struct {
	providers []Provider
	client    *http.Client
	logger    *slog.Logger
}

// NewCommand creates a new Command with the given providers and optional logger.
// Pass nil for logger to disable logging.
func NewCommand(providers []Provider, logger *slog.Logger) *Command {
	return NewCommandWithTimeout(providers, logger, defaultTimeout)
}

// NewCommandWithTimeout creates a new Command with a custom timeout.
func NewCommandWithTimeout(providers []Provider, logger *slog.Logger, timeout time.Duration) *Command {
	return &Command{
		providers: providers,
		client:    &http.Client{Timeout: timeout},
		logger:    logger,
	}
}

// log logs a message if logger is configured.
func (c *Command) log(level slog.Level, msg string, args ...any) {
	if c.logger != nil {
		c.logger.Log(nil, level, msg, args...)
	}
}
