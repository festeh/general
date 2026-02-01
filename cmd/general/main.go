package main

import (
	"bufio"
	"encoding/json"
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
	"kimi":     general.Kimi,
	"gemini":     general.Gemini,
}

var envVarNames = map[string]string{
	"openrouter": "OPENROUTER_API_KEY",
	"groq":       "GROQ_API_KEY",
	"kimi":     "KIMI_API_KEY",
	"gemini":     "GEMINI_API_KEY",
}

func main() {
	var targets targetFlag
	var jsonMode bool
	var rawOutput bool
	flag.Var(&targets, "target", "Target in format provider:model (can be repeated)")
	flag.Var(&targets, "t", "Target in format provider:model (shorthand)")
	flag.BoolVar(&jsonMode, "json", false, "Read ChatCompletionRequest JSON from stdin")
	flag.BoolVar(&rawOutput, "raw", false, "Output full response JSON instead of just content")
	flag.Parse()

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one --target (-t) required")
		fmt.Fprintln(os.Stderr, "Usage: general -t provider:model [-t provider:model ...] [prompt]")
		fmt.Fprintln(os.Stderr, "Providers: openrouter, groq, kimi, gemini")
		fmt.Fprintln(os.Stderr, "API keys from env: OPENROUTER_API_KEY, GROQ_API_KEY, KIMI_API_KEY, GEMINI_API_KEY")
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
			fmt.Fprintln(os.Stderr, "Available: openrouter, groq, kimi, gemini")
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

	// Build request
	var req general.ChatCompletionRequest

	if jsonMode {
		// JSON mode: read ChatCompletionRequest from stdin
		if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid JSON: %v\n", err)
			os.Exit(1)
		}
		if len(req.Messages) == 0 {
			fmt.Fprintln(os.Stderr, "Error: messages array required")
			os.Exit(1)
		}
	} else {
		// Text mode: prompt from args or interactive stdin
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

		req = general.ChatCompletionRequest{
			Messages: []general.ChatCompletionMessage{
				{Role: "user", Content: general.TextContent(prompt)},
			},
		}
	}

	// Execute
	cmd := general.NewCommand(generalTargets, nil)

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

		if rawOutput {
			// Output full response as JSON
			respJSON, _ := json.MarshalIndent(result.Response, "", "  ")
			fmt.Printf("\n[%s] [%s] ✓ %s/%s:\n%s\n",
				timestamp, elapsed,
				providerNameFromEndpoint(result.Target.Provider.Endpoint),
				result.Target.Model,
				string(respJSON),
			)
		} else {
			content := ""
			if len(result.Response.Choices) > 0 {
				content = result.Response.Choices[0].Message.Content.String()
			}
			fmt.Printf("\n[%s] [%s] ✓ %s/%s:\n%s\n",
				timestamp, elapsed,
				providerNameFromEndpoint(result.Target.Provider.Endpoint),
				result.Target.Model,
				content,
			)
		}
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
	case strings.Contains(endpoint, "kimi"):
		return "kimi"
	case strings.Contains(endpoint, "generativelanguage.googleapis"):
		return "gemini"
	default:
		return "unknown"
	}
}
