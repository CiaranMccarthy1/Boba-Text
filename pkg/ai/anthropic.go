package ai

import (
	"encoding/json"
	"fmt"
)

type anthropicRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Messages  []chatMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// anthropicEndpoint is a var so tests can point it at a local server.
var anthropicEndpoint = "https://api.anthropic.com/v1/messages"

// anthropic calls the Anthropic (Claude) messages API.
func anthropic(model string, prompt string) (string, error) {
	apiKey, err := requireKey("ANTHROPIC_API_KEY")
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(anthropicRequest{
		Model:     model,
		MaxTokens: 4096,
		Messages:  []chatMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	respBytes, err := post(anthropicEndpoint, map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": "2023-06-01",
	}, body)
	if err != nil {
		return "", err
	}

	var r anthropicResponse
	if err := decodeJSON(respBytes, &r); err != nil {
		return "", err
	}
	if r.Error != nil {
		return "", fmt.Errorf("API error: %s", r.Error.Message)
	}
	if len(r.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return r.Content[0].Text, nil
}
