package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
)

type githubTransport interface {
	raw(ctx context.Context, c *Client, method, path string, body interface{}, expected ...int) (json.RawMessage, error)
	binaryUpload(ctx context.Context, c *Client, reqURL string, contentType string, body []byte, expected ...int) (json.RawMessage, error)
	downloadBytes(ctx context.Context, c *Client, reqURL string, accept string, expected ...int) ([]byte, error)
}

type ghCLITransport struct {
	binary string
}

func (t ghCLITransport) raw(ctx context.Context, _ *Client, method, path string, body interface{}, _ ...int) (json.RawMessage, error) {
	data, err := marshalCLIRequestBody(body)
	if err != nil {
		return nil, err
	}
	stdout, err := t.run(ctx, cliAPIArgs(method, path, "application/vnd.github+json", "application/json", len(data) > 0), data)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(stdout)) == 0 {
		return nil, nil
	}
	return json.RawMessage(stdout), nil
}

func (t ghCLITransport) binaryUpload(ctx context.Context, _ *Client, reqURL string, contentType string, body []byte, _ ...int) (json.RawMessage, error) {
	stdout, err := t.run(ctx, cliAPIArgs(httpMethodPost, reqURL, "application/vnd.github+json", firstNonEmpty(contentType, "application/octet-stream"), true), body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(stdout)) == 0 {
		return nil, nil
	}
	return json.RawMessage(stdout), nil
}

func (t ghCLITransport) downloadBytes(ctx context.Context, _ *Client, reqURL string, accept string, _ ...int) ([]byte, error) {
	return t.run(ctx, cliAPIArgs(httpMethodGet, reqURL, firstNonEmpty(accept, "application/octet-stream"), "", false), nil)
}

func (t ghCLITransport) run(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
	binary := strings.TrimSpace(t.binary)
	if binary == "" {
		binary = "gh"
	}
	cmd, err := ghCommand(ctx, binary, args...)
	if err != nil {
		return nil, err
	}
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		text := strings.TrimSpace(stderr.String())
		if text == "" {
			text = strings.TrimSpace(err.Error())
		}
		return nil, fmt.Errorf("gh api failed: %s", text)
	}
	return stdout, nil
}

func ghCommand(ctx context.Context, binary string, args ...string) (*exec.Cmd, error) {
	if ext := strings.ToLower(filepath.Ext(binary)); ext == ".cmd" || ext == ".bat" {
		quoted := make([]string, 0, len(args)+2)
		quoted = append(quoted, binary)
		quoted = append(quoted, args...)
		return exec.CommandContext(ctx, "cmd", "/c", strings.Join(quoteArgs(quoted), " ")), nil
	}
	return exec.CommandContext(ctx, binary, args...), nil
}

func cliAPIArgs(method, endpoint, accept, contentType string, withInput bool) []string {
	args := []string{
		"api",
		normalizeCLIEndpoint(endpoint),
		"--method", strings.ToUpper(strings.TrimSpace(method)),
		"-H", "Accept: " + firstNonEmpty(accept, "application/vnd.github+json"),
		"-H", "X-GitHub-Api-Version: 2022-11-28",
	}
	if strings.TrimSpace(contentType) != "" {
		args = append(args, "-H", "Content-Type: "+contentType)
	}
	if withInput {
		args = append(args, "--input", "-")
	}
	return args
}

func normalizeCLIEndpoint(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		if parsed, err := url.Parse(path); err == nil {
			if strings.Contains(parsed.Host, "github") {
				endpoint := strings.TrimPrefix(parsed.Path, "/")
				if parsed.RawQuery != "" {
					endpoint += "?" + parsed.RawQuery
				}
				return endpoint
			}
		}
	}
	return strings.TrimPrefix(path, "/")
}

func marshalCLIRequestBody(body interface{}) ([]byte, error) {
	switch typed := body.(type) {
	case nil:
		return nil, nil
	case json.RawMessage:
		return append([]byte(nil), typed...), nil
	case []byte:
		return append([]byte(nil), typed...), nil
	default:
		return json.Marshal(body)
	}
}

func quoteArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.ReplaceAll(arg, `"`, `\"`)
		if strings.ContainsAny(arg, " \t") {
			arg = `"` + arg + `"`
		}
		out = append(out, arg)
	}
	return out
}

const (
	httpMethodGet  = "GET"
	httpMethodPost = "POST"
)
