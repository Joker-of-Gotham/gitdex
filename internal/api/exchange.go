package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// ExchangeFormat represents supported exchange formats.
type ExchangeFormat string

const (
	ExchangeFormatJSON         ExchangeFormat = "json"
	ExchangeFormatYAML         ExchangeFormat = "yaml"
	ExchangeFormatProtobufJSON ExchangeFormat = "protobuf_json"
)

// ExchangePayload represents versioned exchange data.
type ExchangePayload struct {
	Format        ExchangeFormat  `json:"format" yaml:"format"`
	APIVersion    string          `json:"api_version" yaml:"api_version"`
	SchemaVersion string          `json:"schema_version" yaml:"schema_version"`
	PayloadType   string          `json:"payload_type" yaml:"payload_type"`
	Data          json.RawMessage `json:"data" yaml:"data"`
	Checksum      string          `json:"checksum,omitempty" yaml:"checksum,omitempty"`
	CreatedAt     time.Time       `json:"created_at" yaml:"created_at"`
}

// ExchangeValidator validates exchange payloads.
type ExchangeValidator interface {
	Validate(payload *ExchangePayload) error
}

// DefaultExchangeValidator validates format, version, and required fields.
type DefaultExchangeValidator struct {
	RequiredAPIVersion    string
	RequiredSchemaVersion string
}

// NewDefaultExchangeValidator creates a validator with default requirements.
func NewDefaultExchangeValidator() *DefaultExchangeValidator {
	return &DefaultExchangeValidator{
		RequiredAPIVersion:    "v1",
		RequiredSchemaVersion: "1",
	}
}

// Validate checks payload format, version, and required fields.
func (v *DefaultExchangeValidator) Validate(payload *ExchangePayload) error {
	if payload == nil {
		return fmt.Errorf("payload cannot be nil")
	}
	switch payload.Format {
	case ExchangeFormatJSON, ExchangeFormatYAML, ExchangeFormatProtobufJSON:
		// valid
	default:
		return fmt.Errorf("invalid format %q", payload.Format)
	}
	if strings.TrimSpace(payload.APIVersion) == "" {
		return fmt.Errorf("api_version is required")
	}
	if v.RequiredAPIVersion != "" && payload.APIVersion != v.RequiredAPIVersion {
		return fmt.Errorf("api_version must be %q, got %q", v.RequiredAPIVersion, payload.APIVersion)
	}
	if strings.TrimSpace(payload.SchemaVersion) == "" {
		return fmt.Errorf("schema_version is required")
	}
	if strings.TrimSpace(payload.PayloadType) == "" {
		return fmt.Errorf("payload_type is required")
	}
	return nil
}

// ReadExchangeFile reads an exchange payload from a file (auto-detects format from extension).
func ReadExchangeFile(path string) (*ExchangePayload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	format := ExchangeFormatJSON
	if strings.HasSuffix(strings.ToLower(path), ".yaml") || strings.HasSuffix(strings.ToLower(path), ".yml") {
		format = ExchangeFormatYAML
	}
	return ParseExchangePayload(data, format)
}

// ReadExchangeFileWithFormat reads an exchange payload from a file with explicit format.
func ReadExchangeFileWithFormat(path string, format ExchangeFormat) (*ExchangePayload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return ParseExchangePayload(data, format)
}

// ParseExchangePayload parses raw bytes into ExchangePayload.
func ParseExchangePayload(data []byte, format ExchangeFormat) (*ExchangePayload, error) {
	var p ExchangePayload
	switch format {
	case ExchangeFormatJSON:
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("parse json: %w", err)
		}
	case ExchangeFormatYAML:
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("parse yaml: %w", err)
		}
	default:
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("parse: %w", err)
		}
	}
	if p.Format == "" {
		p.Format = format
	}
	return &p, nil
}
