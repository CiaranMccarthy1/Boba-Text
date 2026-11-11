package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"

// geminiMaxFileBytes caps how much file content is included in one prompt.
const geminiMaxFileBytes = 50 * 1024

// geminiRequest is the request body for the Gemini API.
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// geminiResponse is the response body from the Gemini API.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// sendToGemini creates a tea.Cmd that calls the Gemini API asynchronously.
func sendToGemini(prompt string, currentFile string) tea.Cmd {
	return func() tea.Msg {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return GeminiResponseMsg{
				Err: fmt.Errorf("GEMINI_API_KEY not set. Export it with: set GEMINI_API_KEY=your-key"),
			}
		}

		// Build context-aware prompt
		fullPrompt := prompt
		if currentFile != "" {
			content, err := os.ReadFile(currentFile)
			if err == nil {
				snippet := content
				if len(snippet) > geminiMaxFileBytes {
					snippet = snippet[:geminiMaxFileBytes]
				}
				fullPrompt = fmt.Sprintf(
					"The user is editing the file '%s' with this content:\n```\n%s\n```\n\nUser request: %s\n\nRespond concisely. If suggesting code changes, show the relevant diff or snippet.",
					currentFile, string(snippet), prompt,
				)
			}
		}

		reqBody := geminiRequest{
			Contents: []geminiContent{
				{Parts: []geminiPart{{Text: fullPrompt}}},
			},
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return GeminiResponseMsg{Err: fmt.Errorf("failed to marshal request: %w", err)}
		}

		req, err := http.NewRequest(http.MethodPost, geminiEndpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return GeminiResponseMsg{Err: fmt.Errorf("failed to build request: %w", err)}
		}
		req.Header.Set("Content-Type", "application/json")
		// Header instead of URL query so the key can't leak into proxy/server logs.
		req.Header.Set("x-goog-api-key", apiKey)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return GeminiResponseMsg{Err: fmt.Errorf("API request failed: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return GeminiResponseMsg{Err: fmt.Errorf("Gemini API HTTP %d", resp.StatusCode)}
		}

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return GeminiResponseMsg{Err: fmt.Errorf("failed to read response: %w", err)}
		}

		var geminiResp geminiResponse
		if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
			return GeminiResponseMsg{Err: fmt.Errorf("failed to parse response: %w", err)}
		}

		if geminiResp.Error != nil {
			return GeminiResponseMsg{Err: fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)}
		}

		if len(geminiResp.Candidates) == 0 ||
			len(geminiResp.Candidates[0].Content.Parts) == 0 {
			return GeminiResponseMsg{Err: fmt.Errorf("empty response from Gemini")}
		}

		text := geminiResp.Candidates[0].Content.Parts[0].Text
		return GeminiResponseMsg{Response: text}
	}
}
