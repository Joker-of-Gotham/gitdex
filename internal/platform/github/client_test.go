package github

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func TestBase64Decode(t *testing.T) {
	got, err := base64Decode("aGVsbG8=")
	if err != nil || got != "hello" {
		t.Fatalf("unexpected decode: %q %v", got, err)
	}
}

func TestDetectPlatform(t *testing.T) {
	client := New("token", "owner", "repo")
	got, err := client.DetectPlatform(context.Background(), "git@github.com:owner/repo.git")
	if err != nil || got != platform.PlatformGitHub {
		t.Fatalf("unexpected detect result: %v %v", got, err)
	}
}

func TestCLIClientUsesGhAPITransport(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "gh.cmd")
	content := "@echo off\r\n" +
		"setlocal EnableDelayedExpansion\r\n" +
		"if \"%1\"==\"api\" goto api\r\n" +
		"if \"%1\"==\"auth\" goto auth\r\n" +
		"echo unsupported %* 1>&2\r\n" +
		"exit /b 1\r\n" +
		":auth\r\n" +
		"if \"%2\"==\"token\" echo gh-token\r\n" +
		"exit /b 0\r\n" +
		":api\r\n" +
		"set endpoint=%2\r\n" +
		"if \"%endpoint%\"==\"repos/owner/repo/pages\" (\r\n" +
		"  echo {\"url\":\"https://owner.github.io/repo\",\"status\":\"built\"}\r\n" +
		"  exit /b 0\r\n" +
		")\r\n" +
		"echo unexpected endpoint %endpoint% 1>&2\r\n" +
		"exit /b 1\r\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}

	client := NewCLI(script, "owner", "repo")
	executor := client.AdminExecutors()["pages"]
	snap, err := executor.Inspect(context.Background(), platform.AdminInspectRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if snap == nil || len(snap.State) == 0 {
		t.Fatalf("expected snapshot state from gh api transport")
	}
}
