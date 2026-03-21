package integration_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "gitdex-integration-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.Setenv("GITDEX_USER_CONFIG_DIR", tmpDir)
	_ = os.Unsetenv("GITDEX_CONFIG")
	_ = os.Unsetenv("GITDEX_GLOBAL_CONFIG")
	_ = os.Unsetenv("GITDEX_REPO_CONFIG")
	_ = os.Unsetenv("GITDEX_STORAGE_TYPE")
	_ = os.Unsetenv("GITDEX_STORAGE_DSN")
	_ = os.Setenv("GITDEX_DISABLE_RUNTIME_GITHUB_AUTH", "1")
	_ = os.Unsetenv("GITHUB_TOKEN")
	_ = os.Unsetenv("GH_TOKEN")
	_ = os.Unsetenv("GH_ENTERPRISE_TOKEN")
	_ = os.Unsetenv("GITHUB_ENTERPRISE_TOKEN")

	os.Exit(m.Run())
}
