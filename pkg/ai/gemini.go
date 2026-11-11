package ai

import (
	"encoding/json"
	"fmt"
)

// geminiRequest/geminiResponse is the schema for the Gemini API.
type geminiRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

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

// gemini calls the Google Gemini generateContent API.
func gemini(model string, prompt string) (string, error) {
	apiKey, err := requireKey("GEMINI_API_KEY")
	if err != nil {
		return "", err
	}

	if model == "" || model == "default" {
		model = "gemini-2.0-flash"
	}

	body, err := json.Marshal(geminiRequest{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{{Parts: []struct {
			Text string `json:"text"`
		}{{Text: prompt}}}},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Header instead of URL query so the key can't leak into proxy/server logs.
	respBytes, err := post("https://generativelanguage.googleapis.com/v1beta/models/"+model+":generateContent",
		map[string]string{"x-goog-api-key": apiKey}, body)
	if err != nil {
		return "", err
	}

	var r geminiResponse
	if err := decodeJSON(respBytes, &r); err != nil {
		return "", err
	}
	if r.Error != nil {
		return "", fmt.Errorf("API error: %s", r.Error.Message)
	}
	if len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return r.Candidates[0].Content.Parts[0].Text, nil
}
