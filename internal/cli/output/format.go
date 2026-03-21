package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	FormatText = "text"
	FormatJSON = "json"
	FormatYAML = "yaml"
)

func Normalize(format string) string {
	normalized := strings.TrimSpace(strings.ToLower(format))
	if normalized == "" {
		return FormatText
	}
	return normalized
}

func IsStructured(format string) bool {
	switch Normalize(format) {
	case FormatJSON, FormatYAML:
		return true
	default:
		return false
	}
}

func Marshal(format string, value any) ([]byte, error) {
	switch Normalize(format) {
	case FormatJSON:
		content, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal json output: %w", err)
		}
		return append(content, '\n'), nil
	case FormatYAML:
		content, err := yaml.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal yaml output: %w", err)
		}
		if len(content) == 0 || content[len(content)-1] != '\n' {
			content = append(content, '\n')
		}
		return content, nil
	default:
		return nil, fmt.Errorf("unsupported structured output format %q", format)
	}
}

func WriteValue(w io.Writer, format string, value any) error {
	content, err := Marshal(format, value)
	if err != nil {
		return err
	}

	_, err = w.Write(content)
	return err
}
