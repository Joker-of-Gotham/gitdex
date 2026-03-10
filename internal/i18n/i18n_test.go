package i18n

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalesLoadRequiredKeysWithoutMojibake(t *testing.T) {
	keys := []string{
		"app.name",
		"ui.loading",
		"status_bar.format",
		"language_select.title",
		"model_select.title",
		"input.empty",
		"goal.title",
		"workflow_menu.title",
		"analysis.title",
		"analysis.analyzing",
		"oplog.title",
		"areas.title",
		"thinking.title",
		"suggestions.title",
		"suggestions.keybinds",
		"observability.title",
		"observability.tab_workflow",
		"observability.timeline_title",
		"observability.result_none",
	}

	for _, lang := range []string{"en", "zh", "ja"} {
		t.Run(lang, func(t *testing.T) {
			require.NoError(t, Init(lang))
			assert.Equal(t, lang, Lang())
			for _, key := range keys {
				value := T(key)
				assert.NotEmpty(t, value, key)
				assert.NotEqual(t, key, value, key)
				assert.False(t, hasMojibake(value), "%s -> %q", key, value)
			}
		})
	}
}

func TestLocalesKeepStableKnownTranslations(t *testing.T) {
	cases := map[string]map[string]string{
		"en": {
			"language_select.title": "Choose interface language",
			"analysis.title":        "AI analysis",
			"suggestions.advisory":  "(view only; no command will run; press y to mark reviewed)",
		},
		"zh": {
			"language_select.title": "选择界面语言",
			"analysis.title":        "AI 分析",
			"suggestions.advisory":  "(仅查看；不会执行命令；按 y 标记为已查看)",
		},
		"ja": {
			"language_select.title": "表示言語を選択",
			"analysis.title":        "AI 分析",
			"suggestions.advisory":  "(閲覧のみ。コマンドは実行されません。y で確認済みにします)",
		},
	}

	for lang, expectations := range cases {
		t.Run(lang, func(t *testing.T) {
			require.NoError(t, Init(lang))
			for key, expected := range expectations {
				assert.Equal(t, expected, T(key), key)
			}
		})
	}
}

func hasMojibake(text string) bool {
	if !utf8.ValidString(text) {
		return true
	}
	return strings.ContainsRune(text, '\uFFFD')
}
