package ai

import (
	"encoding/json"
	"fmt"
)

// chatRequest/chatResponse is the schema shared by OpenAI-compatible APIs
// (OpenAI and OpenRouter).
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// chatCompletions calls an OpenAI-compatible chat endpoint (OpenAI, OpenRouter).
func chatCompletions(url string, envVar string, model string, prompt string) (string, error) {
	apiKey, err := requireKey(envVar)
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(chatRequest{
		Model:    model,
		Messages: []chatMessage{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	respBytes, err := post(url, map[string]string{"Authorization": "Bearer " + apiKey}, body)
	if err != nil {
		return "", err
	}

	var r chatResponse
	if err := decodeJSON(respBytes, &r); err != nil {
		return "", err
	}
	if r.Error != nil {
		return "", fmt.Errorf("API error: %s", r.Error.Message)
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return r.Choices[0].Message.Content, nil
}
