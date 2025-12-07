package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/festeh/general"
)

type targetFlag []string

func (t *targetFlag) String() string {
	return strings.Join(*t, ", ")
}

func (t *targetFlag) Set(value string) error {
	*t = append(*t, value)
	return nil
}

var providerConstructors = map[string]func(string) general.Provider{
	"openrouter": general.OpenRouter,
	"groq":       general.Groq,
	"chutes":     general.Chutes,
	"gemini":     general.Gemini,
}

var envVarNames = map[string]string{
	"openrouter": "OPENROUTER_API_KEY",
	"groq":       "GROQ_API_KEY",
	"chutes":     "CHUTES_API_KEY",
	"gemini":     "GEMINI_API_KEY",
}

func main() {
	var targets targetFlag
	flag.Var(&targets, "target", "Target in format provider:model (can be repeated)")
	flag.Var(&targets, "t", "Target in format provider:model (shorthand)")
	flag.Parse()

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one --target (-t) required")
		fmt.Fprintln(os.Stderr, "Usage: general -t provider:model [-t provider:model ...] [prompt]")
		fmt.Fprintln(os.Stderr, "Providers: openrouter, groq, chutes, gemini")
		fmt.Fprintln(os.Stderr, "API keys from env: OPENROUTER_API_KEY, GROQ_API_KEY, CHUTES_API_KEY, GEMINI_API_KEY")
		os.Exit(1)
	}

	// Parse targets
	var generalTargets []general.Target
	for _, t := range targets {
		parts := strings.SplitN(t, ":", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Error: invalid target format %q, expected provider:model\n", t)
			os.Exit(1)
		}

		providerName := strings.ToLower(parts[0])
		model := parts[1]

		constructor, ok := providerConstructors[providerName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: unknown provider %q\n", providerName)
			fmt.Fprintln(os.Stderr, "Available: openrouter, groq, chutes, gemini")
			os.Exit(1)
		}

		envVar := envVarNames[providerName]
		apiKey := os.Getenv(envVar)
		if apiKey == "" {
			fmt.Fprintf(os.Stderr, "Error: %s not set\n", envVar)
			os.Exit(1)
		}

		provider := constructor(apiKey)
		generalTargets = append(generalTargets, general.Target{
			Provider: provider,
			Model:    model,
		})
	}

	// Get prompt from args or stdin
	var prompt string
	if flag.NArg() > 0 {
		prompt = strings.Join(flag.Args(), " ")
	} else {
		fmt.Fprintln(os.Stderr, "Enter prompt (Ctrl+D to send):")
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		prompt = strings.Join(lines, "\n")
	}

	if strings.TrimSpace(prompt) == "" {
		fmt.Fprintln(os.Stderr, "Error: empty prompt")
		os.Exit(1)
	}

	// Execute
	cmd := general.NewCommand(generalTargets, nil)
	req := general.ChatCompletionRequest{
		Messages: []general.ChatCompletionMessage{
			{Role: "user", Content: prompt},
		},
	}

	startTime := time.Now()
	fmt.Fprintf(os.Stderr, "[%s] Sending to %d target(s)...\n", startTime.Format("15:04:05.000"), len(generalTargets))

	results := cmd.Execute(req)

	for result := range results {
		timestamp := time.Now().Format("15:04:05.000")
		elapsed := time.Since(startTime).Round(time.Millisecond)

		if result.Error != nil {
			fmt.Printf("[%s] [%s] ❌ %s/%s: %v\n",
				timestamp, elapsed,
				providerNameFromEndpoint(result.Target.Provider.Endpoint),
				result.Target.Model,
				result.Error,
			)
			continue
		}

		content := ""
		if len(result.Response.Choices) > 0 {
			content = result.Response.Choices[0].Message.Content
		}

		fmt.Printf("\n[%s] [%s] ✓ %s/%s:\n%s\n",
			timestamp, elapsed,
			providerNameFromEndpoint(result.Target.Provider.Endpoint),
			result.Target.Model,
			content,
		)
	}

	fmt.Fprintf(os.Stderr, "\n[%s] Done (total: %s)\n",
		time.Now().Format("15:04:05.000"),
		time.Since(startTime).Round(time.Millisecond),
	)
}

func providerNameFromEndpoint(endpoint string) string {
	switch {
	case strings.Contains(endpoint, "openrouter"):
		return "openrouter"
	case strings.Contains(endpoint, "groq"):
		return "groq"
	case strings.Contains(endpoint, "chutes"):
		return "chutes"
	case strings.Contains(endpoint, "generativelanguage.googleapis"):
		return "gemini"
	default:
		return "unknown"
	}
}
