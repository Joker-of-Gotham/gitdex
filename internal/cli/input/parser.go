package input

import (
	"strings"

	"github.com/spf13/cobra"
)

type InputType int

const (
	InputCommand InputType = iota
	InputNaturalLanguage
	InputEmpty
)

func (t InputType) String() string {
	switch t {
	case InputCommand:
		return "command"
	case InputNaturalLanguage:
		return "natural_language"
	case InputEmpty:
		return "empty"
	default:
		return "unknown"
	}
}

type ClassifiedInput struct {
	Type    InputType
	Raw     string
	Command string
	Args    []string
}

type Parser struct {
	rootCmd *cobra.Command
}

func NewParser(rootCmd *cobra.Command) *Parser {
	return &Parser{rootCmd: rootCmd}
}

func (p *Parser) Classify(raw string) ClassifiedInput {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ClassifiedInput{Type: InputEmpty, Raw: raw}
	}

	tokens := strings.Fields(trimmed)

	cmd, remaining, err := p.rootCmd.Find(tokens)
	if err == nil && cmd != nil && cmd != p.rootCmd {
		return ClassifiedInput{
			Type:    InputCommand,
			Raw:     raw,
			Command: cmd.CommandPath(),
			Args:    remaining,
		}
	}

	return ClassifiedInput{
		Type: InputNaturalLanguage,
		Raw:  raw,
	}
}
