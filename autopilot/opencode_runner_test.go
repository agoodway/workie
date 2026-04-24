package autopilot

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpencodeRunnerRun(t *testing.T) {
	var deleteCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			if got := r.Header.Get("x-opencode-directory"); got != "/project" {
				t.Fatalf("expected x-opencode-directory /project, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"session-1"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/session/session-1/message":
			if got := r.Header.Get("x-opencode-directory"); got != "/project" {
				t.Fatalf("expected x-opencode-directory /project, got %q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("expected valid JSON body: %v", err)
			}
			model := body["model"].(map[string]any)
			if model["providerID"] != "openai" || model["modelID"] != "gpt-5.5" {
				t.Fatalf("unexpected model payload: %#v", model)
			}
			parts := body["parts"].([]any)
			part := parts[0].(map[string]any)
			if part["text"] != "prompt text" {
				t.Fatalf("unexpected prompt payload: %#v", part)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"parts":[{"type":"text","text":"hello from server"}]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/session/session-1":
			if got := r.Header.Get("x-opencode-directory"); got != "/project" {
				t.Fatalf("expected x-opencode-directory /project, got %q", got)
			}
			deleteCalled = true
			_, _ = w.Write([]byte(`true`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	runner := newOpencodeRunner(PipelineOptions{OpencodeServerURL: server.URL}).(*opencodeRunner)
	var stdout bytes.Buffer
	if err := runner.Run(context.Background(), SessionRequest{Prompt: "prompt text", Model: "openai/gpt-5.5", WorkingDir: "/project"}, &stdout, &stdout); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stdout.String() != "hello from server\n" {
		t.Fatalf("expected server response in stdout, got %q", stdout.String())
	}
	if !deleteCalled {
		t.Fatal("expected session cleanup to run")
	}
}

func TestOpencodeRunnerPreflightChecksWorkspace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/global/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"healthy":true}`))
		case "/path":
			if got := r.URL.Query().Get("directory"); got != "/expected/project" {
				t.Fatalf("expected directory query /expected/project, got %q", got)
			}
			if got := r.Header.Get("x-opencode-directory"); got != "/expected/project" {
				t.Fatalf("expected x-opencode-directory /expected/project, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"directory":"/somewhere/else"}`))
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	runner := newOpencodeRunner(PipelineOptions{OpencodeServerURL: server.URL}).(*opencodeRunner)
	err := runner.Preflight(context.Background(), SessionRequest{Prompt: "say ok", Model: "openai/gpt-5.5", WorkingDir: "/expected/project"})
	if err == nil {
		t.Fatal("expected workspace mismatch error")
	}
	if !strings.Contains(err.Error(), "/expected/project") {
		t.Fatalf("expected working directory in error, got %v", err)
	}
}

func TestOpencodeRunnerPreflightAuthHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`unauthorized`))
	}))
	defer server.Close()

	runner := newOpencodeRunner(PipelineOptions{OpencodeServerURL: server.URL}).(*opencodeRunner)
	err := runner.Preflight(context.Background(), SessionRequest{Prompt: "say ok", Model: "openai/gpt-5.5", WorkingDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if !strings.Contains(err.Error(), DefaultOpencodeServerPasswordEnv) {
		t.Fatalf("expected auth guidance in error, got %v", err)
	}
}

func TestOpencodeRunnerSendsBasicAuth(t *testing.T) {
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("opencode:secret"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != expectedAuth {
			t.Fatalf("expected basic auth header %q, got %q", expectedAuth, got)
		}
		switch r.URL.Path {
		case "/global/health":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"healthy":true}`))
		case "/path":
			if got := r.URL.Query().Get("directory"); got != "/project" {
				t.Fatalf("expected directory query /project, got %q", got)
			}
			if got := r.Header.Get("x-opencode-directory"); got != "/project" {
				t.Fatalf("expected x-opencode-directory /project, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"directory":"/project"}`))
		case "/session":
			if got := r.Header.Get("x-opencode-directory"); got != "/project" {
				t.Fatalf("expected x-opencode-directory /project, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"session-1"}`))
		case "/session/session-1/message":
			if got := r.Header.Get("x-opencode-directory"); got != "/project" {
				t.Fatalf("expected x-opencode-directory /project, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"parts":[{"type":"text","text":"ok"}]}`))
		case "/session/session-1":
			if got := r.Header.Get("x-opencode-directory"); got != "/project" {
				t.Fatalf("expected x-opencode-directory /project, got %q", got)
			}
			_, _ = w.Write([]byte(`true`))
		default:
			t.Fatalf("unexpected request path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	runner := newOpencodeRunner(PipelineOptions{
		OpencodeServerURL:      server.URL,
		OpencodeServerUsername: "opencode",
		OpencodeServerPassword: "secret",
	}).(*opencodeRunner)

	if err := runner.Preflight(context.Background(), SessionRequest{Prompt: "say ok", Model: "openai/gpt-5.5", WorkingDir: "/project"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
