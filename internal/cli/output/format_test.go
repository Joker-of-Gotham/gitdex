package output_test

import (
	"bytes"
	"testing"

	"github.com/your-org/gitdex/internal/cli/output"
)

func TestFormatsAreStableAndDistinct(t *testing.T) {
	formats := map[string]bool{
		output.FormatText: false,
		output.FormatJSON: false,
		output.FormatYAML: false,
	}

	if len(formats) != 3 {
		t.Fatalf("expected 3 distinct formats, got %d", len(formats))
	}

	for format := range formats {
		if format == "" {
			t.Fatal("format should not be empty")
		}
	}
}

func TestNormalizeDefaultsToText(t *testing.T) {
	if got := output.Normalize(""); got != output.FormatText {
		t.Fatalf("Normalize(\"\") = %q, want %q", got, output.FormatText)
	}
}

func TestStructuredFormatsMarshal(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   string
	}{
		{name: "json", format: output.FormatJSON, want: "\"status\": \"ok\""},
		{name: "yaml", format: output.FormatYAML, want: "status: ok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := output.Marshal(tt.format, map[string]string{"status": "ok"})
			if err != nil {
				t.Fatalf("Marshal returned error: %v", err)
			}

			if !bytes.Contains(content, []byte(tt.want)) {
				t.Fatalf("Marshal(%q) = %q, want substring %q", tt.format, string(content), tt.want)
			}
		})
	}
}

func TestWriteValueRejectsTextFormat(t *testing.T) {
	var out bytes.Buffer

	err := output.WriteValue(&out, output.FormatText, map[string]string{"status": "ok"})
	if err == nil {
		t.Fatal("expected error for text format")
	}
}
