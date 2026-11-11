// Package ai sends prompts (with optional conversation history and file
// context) to the configured LLM provider (Gemini, OpenAI, Claude/Anthropic or
// OpenRouter) and returns the reply text.
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// maxFileBytes caps how much file content is included in one prompt.
const maxFileBytes = 50 * 1024

// Send sends prompt plus the conversation so far and, when needed, the
// current buffer's content to the provider.
func Send(provider, model, fileName, content, prompt string, history []string) (string, error) {
	fullPrompt := buildPrompt(fileName, content, prompt, history)

	switch strings.ToLower(provider) {
	case "openai":
		return chatCompletions("https://api.openai.com/v1/chat/completions", "OPENAI_API_KEY", model, fullPrompt)
	case "openrouter":
		return chatCompletions("https://openrouter.ai/api/v1/chat/completions", "OPENROUTER_API_KEY", model, fullPrompt)
	case "claude", "anthropic":
		return anthropic(model, fullPrompt)
	case "gemini", "":
		return gemini(model, fullPrompt)
	default:
		return "", fmt.Errorf("unknown AI provider %q (use gemini, openai, claude or openrouter)", provider)
	}
}

// buildPrompt prepends the conversation history and the buffer's content
// (capped) to the user's prompt.
func buildPrompt(fileName, content, prompt string, history []string) string {
	var b strings.Builder
	if len(history) > 0 {
		// shortcut: history is sent as plain text, not provider-native message
		// arrays; upgrade if role separation or per-message token accounting matters.
		b.WriteString("Conversation so far:\n")
		for _, turn := range history {
			b.WriteString(turn)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if fileName == "" {
		b.WriteString(prompt)
		return b.String()
	}
	if len(content) > maxFileBytes {
		content = content[:maxFileBytes]
	}
	fmt.Fprintf(&b,
		"The user is editing the file '%s' with this content:\n```\n%s\n```\n\nUser request: %s\n\nRespond concisely. If suggesting code changes, show the relevant diff or snippet.",
		fileName, content, prompt,
	)
	return b.String()
}

// post sends body as JSON with headers and returns the response body,
// failing fast on non-200 so proxy/HTML errors don't confuse JSON parsing.
func post(url string, headers map[string]string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %.200s", resp.StatusCode, respBytes)
	}
	return respBytes, nil
}

// requireKey fetches an API key from the environment or fails with a setup hint.
func requireKey(envVar string) (string, error) {
	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		return "", fmt.Errorf("%s not set", envVar)
	}
	return apiKey, nil
}

// decodeJSON unmarshals the response body into v.
func decodeJSON(respBytes []byte, v any) error {
	if err := json.Unmarshal(respBytes, v); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	return nil
}
