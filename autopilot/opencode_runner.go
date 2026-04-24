package autopilot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"path/filepath"
	"strings"
)

type opencodeRunner struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

type opencodeHTTPError struct {
	statusCode int
	body       string
}

func (e *opencodeHTTPError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("opencode server returned HTTP %d", e.statusCode)
	}
	return fmt.Sprintf("opencode server returned HTTP %d: %s", e.statusCode, e.body)
}

type opencodeSession struct {
	ID string `json:"id"`
}

type opencodeSessionCreateRequest struct {
	Title string `json:"title,omitempty"`
}

type opencodePathResponse struct {
	Path      string `json:"path,omitempty"`
	Current   string `json:"current,omitempty"`
	Root      string `json:"root,omitempty"`
	Directory string `json:"directory,omitempty"`
	Worktree  string `json:"worktree,omitempty"`
	Home      string `json:"home,omitempty"`
}

type opencodeMessageModel struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

type opencodeMessagePart struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Content string `json:"content,omitempty"`
}

type opencodeMessageRequest struct {
	Model opencodeMessageModel  `json:"model"`
	Parts []opencodeMessagePart `json:"parts"`
}

type opencodeMessageResponse struct {
	Parts []opencodeMessagePart `json:"parts"`
}

type opencodeHealthResponse struct {
	Healthy bool   `json:"healthy"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

func newOpencodeRunner(opts PipelineOptions) SessionRunner {
	baseURL := strings.TrimRight(strings.TrimSpace(opts.OpencodeServerURL), "/")
	if baseURL == "" {
		baseURL = DefaultOpencodeServerURL
	}

	username := strings.TrimSpace(opts.OpencodeServerUsername)
	password := strings.TrimSpace(opts.OpencodeServerPassword)
	if password != "" && username == "" {
		username = DefaultOpencodeServerUsername
	}

	return &opencodeRunner{
		baseURL:  baseURL,
		username: username,
		password: password,
		client:   &http.Client{},
	}
}

func (r *opencodeRunner) Preflight(ctx context.Context, req SessionRequest) error {
	if err := r.healthCheck(ctx); err != nil {
		return r.decoratePreflightError(err)
	}
	if err := r.verifyWorkingDir(ctx, req.WorkingDir); err != nil {
		return fmt.Errorf("preflight check failed: %w", err)
	}

	probe := req
	if probe.Prompt == "" {
		probe.Prompt = "say ok"
	}
	if err := r.Run(ctx, probe, io.Discard, io.Discard); err != nil {
		return r.decoratePreflightError(err)
	}

	return nil
}

func (r *opencodeRunner) Run(ctx context.Context, req SessionRequest, stdout, _ io.Writer) (runErr error) {
	session, err := r.createSession(ctx, req.WorkingDir)
	if err != nil {
		return fmt.Errorf("opencode session failed to create: %w", err)
	}

	defer func() {
		if cleanupErr := r.deleteSession(ctx, session.ID, req.WorkingDir); cleanupErr != nil && runErr == nil {
			runErr = fmt.Errorf("opencode session cleanup failed: %w", cleanupErr)
		}
	}()

	resp, err := r.sendMessage(ctx, session.ID, req)
	if err != nil {
		return fmt.Errorf("opencode session failed: %w", err)
	}

	if _, err := io.WriteString(stdout, extractOpencodeResponseText(resp)); err != nil {
		return fmt.Errorf("failed to write opencode response: %w", err)
	}

	return nil
}

func (r *opencodeRunner) healthCheck(ctx context.Context) error {
	var health opencodeHealthResponse
	if err := r.doJSON(ctx, http.MethodGet, "/global/health", nil, &health, ""); err != nil {
		return err
	}
	if !health.Healthy && health.Status != "ok" {
		return fmt.Errorf("opencode server health check failed")
	}
	return nil
}

func (r *opencodeRunner) verifyWorkingDir(ctx context.Context, workingDir string) error {
	if strings.TrimSpace(workingDir) == "" {
		return nil
	}

	var pathResp opencodePathResponse
	if err := r.doJSON(ctx, http.MethodGet, "/path", nil, &pathResp, workingDir); err != nil {
		return err
	}

	cleanProjectDir := filepath.Clean(workingDir)
	for _, candidate := range []string{pathResp.Directory, pathResp.Worktree, pathResp.Path, pathResp.Current, pathResp.Root} {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if filepath.Clean(candidate) == cleanProjectDir {
			return nil
		}
	}

	serverPath := strings.TrimSpace(pathResp.Directory)
	if serverPath == "" {
		serverPath = strings.TrimSpace(pathResp.Worktree)
	}
	if serverPath == "" {
		serverPath = strings.TrimSpace(pathResp.Root)
	}
	if serverPath == "" {
		serverPath = strings.TrimSpace(pathResp.Current)
	}
	if serverPath == "" {
		serverPath = strings.TrimSpace(pathResp.Path)
	}
	if serverPath == "" {
		return nil
	}

	return fmt.Errorf("opencode server resolved %s, expected %s", serverPath, cleanProjectDir)
}

func (r *opencodeRunner) createSession(ctx context.Context, workingDir string) (opencodeSession, error) {
	var session opencodeSession
	err := r.doJSON(ctx, http.MethodPost, "/session", opencodeSessionCreateRequest{Title: "workie autopilot"}, &session, workingDir)
	if err != nil {
		return opencodeSession{}, err
	}
	if strings.TrimSpace(session.ID) == "" {
		return opencodeSession{}, fmt.Errorf("opencode server returned an empty session id")
	}
	return session, nil
}

func (r *opencodeRunner) sendMessage(ctx context.Context, sessionID string, req SessionRequest) (opencodeMessageResponse, error) {
	providerID, modelID := splitOpencodeModel(req.Model)
	body := opencodeMessageRequest{
		Model: opencodeMessageModel{ProviderID: providerID, ModelID: modelID},
		Parts: []opencodeMessagePart{{Type: "text", Text: req.Prompt}},
	}

	var resp opencodeMessageResponse
	err := r.doJSON(ctx, http.MethodPost, fmt.Sprintf("/session/%s/message", sessionID), body, &resp, req.WorkingDir)
	if err != nil {
		return opencodeMessageResponse{}, err
	}
	return resp, nil
}

func (r *opencodeRunner) deleteSession(ctx context.Context, sessionID, workingDir string) error {
	return r.doJSON(ctx, http.MethodDelete, fmt.Sprintf("/session/%s", sessionID), nil, nil, workingDir)
}

func (r *opencodeRunner) doJSON(ctx context.Context, method, endpoint string, body any, out any, workingDir string) error {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal opencode request: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	requestURL := r.baseURL + endpoint
	if workingDir = strings.TrimSpace(workingDir); workingDir != "" && !strings.HasPrefix(endpoint, "/global/") && (method == http.MethodGet || method == http.MethodHead) {
		parsedURL, err := urlpkg.Parse(requestURL)
		if err != nil {
			return fmt.Errorf("failed to build opencode request URL: %w", err)
		}
		query := parsedURL.Query()
		query.Set("directory", workingDir)
		parsedURL.RawQuery = query.Encode()
		requestURL = parsedURL.String()
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to build opencode request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if workingDir != "" && !strings.HasPrefix(endpoint, "/global/") {
		req.Header.Set("x-opencode-directory", workingDir)
	}
	if r.password != "" {
		req.SetBasicAuth(r.username, r.password)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		return fmt.Errorf("failed to reach opencode server: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read opencode response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &opencodeHTTPError{statusCode: resp.StatusCode, body: strings.TrimSpace(string(responseBody))}
	}

	if out == nil || len(responseBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode opencode response: %w", err)
	}

	return nil
}

func (r *opencodeRunner) decoratePreflightError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("preflight check failed: opencode server did not respond within 30s")
	}

	var httpErr *opencodeHTTPError
	if errors.As(err, &httpErr) && (httpErr.statusCode == http.StatusUnauthorized || httpErr.statusCode == http.StatusForbidden) {
		return fmt.Errorf(
			"preflight check failed: opencode server authentication failed (set %s/%s or configure autopilot.opencode.username_env/password_env)",
			DefaultOpencodeServerUsernameEnv,
			DefaultOpencodeServerPasswordEnv,
		)
	}

	return fmt.Errorf("preflight check failed: %w", err)
}

func splitOpencodeModel(model string) (string, string) {
	trimmed := strings.TrimSpace(model)
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "openai", trimmed
}

func extractOpencodeResponseText(resp opencodeMessageResponse) string {
	var parts []string
	for _, part := range resp.Parts {
		if text := strings.TrimSpace(part.Text); text != "" {
			parts = append(parts, text)
			continue
		}
		if content := strings.TrimSpace(part.Content); content != "" {
			parts = append(parts, content)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	text := strings.Join(parts, "\n")
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text
}
