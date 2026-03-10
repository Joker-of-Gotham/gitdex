package i18n

import (
	"embed"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localeFS embed.FS

var (
	currentLang string
	messages    map[string]string
	mu          sync.RWMutex
)

func Init(lang string) error {
	mu.Lock()
	defer mu.Unlock()

	if lang == "" || lang == "auto" {
		lang = detectLanguage()
	}
	currentLang = lang

	data, err := localeFS.ReadFile(fmt.Sprintf("locales/%s.yaml", lang))
	if err != nil {
		data, err = localeFS.ReadFile("locales/en.yaml")
		if err != nil {
			return fmt.Errorf("i18n: load fallback locale: %w", err)
		}
		currentLang = "en"
	}

	messages = make(map[string]string)
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("i18n: parse locale: %w", err)
	}
	flatten("", raw, messages)
	return nil
}

func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()
	if msg, ok := messages[key]; ok {
		return msg
	}
	return key
}

func Lang() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

func flatten(prefix string, m map[string]interface{}, out map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case string:
			out[key] = val
		case map[string]interface{}:
			flatten(key, val, out)
		}
	}
}

func detectLanguage() string {
	for _, env := range []string{"LANG", "LC_ALL", "LANGUAGE"} {
		val := os.Getenv(env)
		if val == "" {
			continue
		}
		val = strings.ToLower(val)
		if strings.HasPrefix(val, "zh") {
			return "zh"
		}
		if strings.HasPrefix(val, "ja") {
			return "ja"
		}
		if strings.HasPrefix(val, "en") {
			return "en"
		}
	}
	return "en"
}
