package app

import (
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
)

func TestParseLang(t *testing.T) {
	cases := map[string]string{
		"zh_CN.UTF-8": "zh",
		"ja_JP":       "ja",
		"en_US":       "en",
		"unknown":     "",
	}
	for in, want := range cases {
		if got := parseLang(in); got != want {
			t.Fatalf("parseLang(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCheckLLMEnvironment(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLM.Provider = "openai"
	cfg.LLM.Primary.Provider = "openai"
	cfg.LLM.Primary.Model = "gpt-4.1-mini"
	cfg.LLM.Primary.APIKey = ""
	cfg.LLM.Primary.APIKeyEnv = ""

	status := checkLLMEnvironment(cfg)
	if status == "" {
		t.Fatal("expected environment status")
	}
}
