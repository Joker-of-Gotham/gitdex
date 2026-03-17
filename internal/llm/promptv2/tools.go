package promptv2

import (
	"fmt"
	"strings"
)

// ToolDef describes a single tool available to the LLM planner,
// following MCP (Model Context Protocol) Tool Definition conventions.
type ToolDef struct {
	Name        string
	Description string
	Params      []ToolParam
}

// ToolParam describes one parameter of a tool.
type ToolParam struct {
	Name        string
	Type        string
	Required    bool
	Description string
	Enum        []string
}

// Tools lists all tool types the planner can use in suggestion actions.
var Tools = []ToolDef{
	{
		Name:        "git_command",
		Description: "Execute a git CLI command. One command per suggestion.",
		Params: []ToolParam{
			{Name: "command", Type: "string", Required: true, Description: "Complete git command string, e.g. git fetch --all --prune"},
		},
	},
	{
		Name:        "shell_command",
		Description: "Execute a build/test/utility command (go test, npm run build, etc.). Shell operators (&&, |, ;) are not supported.",
		Params: []ToolParam{
			{Name: "command", Type: "string", Required: true, Description: "Complete command string"},
		},
	},
	{
		Name:        "file_write",
		Description: "Create, update, delete, or append to a file. Parent directories are auto-created.",
		Params: []ToolParam{
			{Name: "file_path", Type: "string", Required: true, Description: "Relative file path"},
			{Name: "file_content", Type: "string", Required: false, Description: "Full file content (for create/update/append)"},
			{Name: "file_operation", Type: "string", Required: true, Description: "Operation type", Enum: []string{"create", "update", "delete", "append", "mkdir"}},
		},
	},
	{
		Name:        "file_read",
		Description: "Read file content. Returns content in stdout.",
		Params: []ToolParam{
			{Name: "file_path", Type: "string", Required: true, Description: "Relative file path to read"},
		},
	},
	{
		Name:        "github_op",
		Description: "Execute a GitHub CLI (gh) command.",
		Params: []ToolParam{
			{Name: "command", Type: "string", Required: true, Description: "Complete gh command, e.g. gh issue create --title \"...\""},
		},
	},
}

// ValidToolTypes returns the set of valid action type strings.
func ValidToolTypes() map[string]bool {
	m := make(map[string]bool, len(Tools))
	for _, t := range Tools {
		m[t.Name] = true
	}
	return m
}

// RenderToolDefs formats tool definitions for injection into system prompts.
func RenderToolDefs() string {
	var b strings.Builder
	b.WriteString("TOOLS (use \"type\" field to select):\n")
	for _, t := range Tools {
		b.WriteString(fmt.Sprintf("\n- \"%s\": %s\n", t.Name, t.Description))
		for _, p := range t.Params {
			req := ""
			if p.Required {
				req = ", required"
			}
			enumStr := ""
			if len(p.Enum) > 0 {
				enumStr = fmt.Sprintf(", enum: [%s]", strings.Join(p.Enum, ", "))
			}
			b.WriteString(fmt.Sprintf("    %s (%s%s%s): %s\n", p.Name, p.Type, req, enumStr, p.Description))
		}
	}
	return b.String()
}
