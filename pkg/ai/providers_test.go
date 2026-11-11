package ai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serve runs handler against a local server for one test.
func serve(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

func checkResult(t *testing.T, got string, err error, want string, wantErr string) {
	t.Helper()
	if wantErr != "" {
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Errorf("got (%q, %v), want error containing %q", got, err, wantErr)
		}
		return
	}
	if err != nil || got != want {
		t.Errorf("got (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestChatCompletions(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test")
	cases := []struct {
		name, body string
		status     int
		want       string
		wantErr    string
	}{
		{"ok", `{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`, 200, "hello", ""},
		{"api error", `{"error":{"message":"bad key"}}`, 200, "", "bad key"},
		{"empty choices", `{"choices":[]}`, 200, "", "empty response"},
		{"http error", `nope`, 500, "", "HTTP 500"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := serve(t, func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer test" {
					t.Errorf("Authorization = %q, want Bearer test", got)
				}
				w.WriteHeader(c.status)
				w.Write([]byte(c.body))
			})
			got, err := chatCompletions(ts.URL, "OPENAI_API_KEY", "m", "hi")
			checkResult(t, got, err, c.want, c.wantErr)
		})
	}
}

func TestAnthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test")
	old := anthropicEndpoint
	t.Cleanup(func() { anthropicEndpoint = old })

	cases := []struct {
		name, body string
		want       string
		wantErr    string
	}{
		{"ok", `{"content":[{"type":"text","text":"hello"}]}`, "hello", ""},
		{"api error", `{"error":{"message":"boom"}}`, "", "boom"},
		{"empty", `{"content":[]}`, "", "empty response"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := serve(t, func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("x-api-key"); got != "test" {
					t.Errorf("x-api-key = %q, want test", got)
				}
				if got := r.Header.Get("anthropic-version"); got == "" {
					t.Error("anthropic-version header missing")
				}
				w.Write([]byte(c.body))
			})
			anthropicEndpoint = ts.URL
			got, err := anthropic("m", "hi")
			checkResult(t, got, err, c.want, c.wantErr)
		})
	}
}

func TestGemini(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test")
	old := geminiEndpoint
	t.Cleanup(func() { geminiEndpoint = old })

	cases := []struct {
		name, body string
		want       string
		wantErr    string
	}{
		{"ok", `{"candidates":[{"content":{"parts":[{"text":"hello"}]}}]}`, "hello", ""},
		{"api error", `{"error":{"message":"boom"}}`, "", "boom"},
		{"empty", `{"candidates":[]}`, "", "empty response"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := serve(t, func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("x-goog-api-key"); got != "test" {
					t.Errorf("x-goog-api-key = %q, want test", got)
				}
				if !strings.Contains(r.URL.Path, "gemini-2.0-flash") {
					t.Errorf("path %q does not contain the default model", r.URL.Path)
				}
				w.Write([]byte(c.body))
			})
			geminiEndpoint = ts.URL + "/%s"
			got, err := gemini("", "hi")
			checkResult(t, got, err, c.want, c.wantErr)
		})
	}
}
