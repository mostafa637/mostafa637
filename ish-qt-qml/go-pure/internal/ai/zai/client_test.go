package zai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(checkChat))
	defer server.Close()
	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, Model: "glm-test"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Choices[0].Message.Content != "ok" {
		t.Fatalf("content=%q", result.Choices[0].Message.Content)
	}
}

func checkChat(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	body, _ := io.ReadAll(r.Body)
	if !strings.Contains(string(body), `"model":"glm-test"`) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("ZAI_API_KEY", "env-key")
	t.Setenv("ZAI_MODEL", "env-model")
	cfg := ConfigFromEnv()
	if cfg.APIKey != "env-key" || cfg.Model != "env-model" {
		t.Fatalf("config=%+v", cfg)
	}
}
