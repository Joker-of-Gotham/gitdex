package contract

import (
	"fmt"
	"strings"
)

var validActionTypes = map[string]bool{
	"git_command":   true,
	"shell_command": true,
	"file_write":    true,
	"file_read":     true,
	"github_op":     true,
}

var validFileOps = map[string]bool{
	"create": true,
	"update": true,
	"append": true,
	"delete": true,
	"mkdir":  true,
}

// ValidateSuggestion validates V3 suggestion contract constraints.
func ValidateSuggestion(s SuggestionItem) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("suggestion.name is required")
	}
	if strings.TrimSpace(s.Reason) == "" {
		return fmt.Errorf("suggestion.reason is required")
	}
	return ValidateAction(s.Action)
}

// ValidateAction validates V3 action contract constraints.
func ValidateAction(a ActionSpec) error {
	typ := strings.TrimSpace(a.Type)
	if !validActionTypes[typ] {
		return fmt.Errorf("action.type %q is invalid", a.Type)
	}

	switch typ {
	case "git_command", "shell_command", "github_op":
		if strings.TrimSpace(a.Command) == "" {
			return fmt.Errorf("action.command is required for %s", typ)
		}
		if hasCommandPlaceholderToken(a.Command) {
			return fmt.Errorf("action.command contains placeholder token; provide a complete command")
		}
	case "file_read":
		if strings.TrimSpace(a.FilePath) == "" {
			return fmt.Errorf("action.file_path is required for file_read")
		}
		if strings.Contains(a.FilePath, "...") {
			return fmt.Errorf("action.file_path contains placeholder; provide a concrete path")
		}
	case "file_write":
		if strings.TrimSpace(a.FilePath) == "" {
			return fmt.Errorf("action.file_path is required for file_write")
		}
		if strings.Contains(a.FilePath, "...") {
			return fmt.Errorf("action.file_path contains placeholder; provide a concrete path")
		}
		op := strings.TrimSpace(a.FileOp)
		if !validFileOps[op] {
			return fmt.Errorf("action.file_operation %q is invalid", a.FileOp)
		}
		if op == "create" || op == "update" || op == "append" {
			if a.FileContent == "" {
				return fmt.Errorf("action.file_content is required for file_operation=%s", op)
			}
		}
	}
	return nil
}

func hasCommandPlaceholderToken(command string) bool {
	for _, tok := range strings.Fields(command) {
		if !strings.Contains(tok, "...") {
			continue
		}
		// Keep common Go package wildcard usage valid.
		if tok == "./..." || tok == "../..." {
			continue
		}
		return true
	}
	return false
}
