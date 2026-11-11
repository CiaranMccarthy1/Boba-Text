package ai

import (
	"strings"
	"testing"
)

func TestSendUnknownProvider(t *testing.T) {
	if _, err := Send("gpt5-nonsense", "x", "", "", "hi"); err == nil || !strings.Contains(err.Error(), "unknown AI provider") {
		t.Errorf("Send with unknown provider: got err=%v, want unknown-provider error", err)
	}
}

func TestSendMissingKey(t *testing.T) {
	cases := []struct{ provider, env string }{
		{"gemini", "GEMINI_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"claude", "ANTHROPIC_API_KEY"},
		{"openrouter", "OPENROUTER_API_KEY"},
	}
	for _, c := range cases {
		t.Setenv(c.env, "")
		if _, err := Send(c.provider, "model", "", "", "hi"); err == nil || !strings.Contains(err.Error(), c.env) {
			t.Errorf("Send(%q) without key: got err=%v, want %q missing error", c.provider, err, c.env)
		}
	}
}

func TestBuildPrompt(t *testing.T) {
	if got := buildPrompt("", "", "hi"); got != "hi" {
		t.Errorf("buildPrompt without file: got %q", got)
	}
	got := buildPrompt("f.txt", strings.Repeat("a", maxFileBytes+10), "hi")
	if !strings.Contains(got, "hi") {
		t.Fatal("buildPrompt: prompt missing")
	}
	// Count only the fenced content (snippet + its trailing newline).
	start := strings.Index(got, "```") + len("```\n")
	end := strings.LastIndex(got, "```")
	if n := end - start; n != maxFileBytes+1 {
		t.Errorf("buildPrompt: content is %d bytes, want capped %d (+1 for trailing newline)", n, maxFileBytes)
	}
}
