package autonomy

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type ToolParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type Tool struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  map[string]ToolParam `json:"parameters"`
	Handler     ActionHandler        `json:"-"`
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) List() []Tool {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]Tool, 0, len(r.tools))
	for _, name := range names {
		out = append(out, r.tools[name])
	}
	return out
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, args map[string]string) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	if tool.Handler == nil {
		return "", fmt.Errorf("tool %s has no handler", name)
	}
	return tool.Handler(ctx, args)
}

func (r *ToolRegistry) GenerateToolPrompt() string {
	var result strings.Builder
	result.WriteString("Available tools:\n\n")
	for _, t := range r.List() {
		result.WriteString(fmt.Sprintf("- %s: %s\n", t.Name, t.Description))
		if len(t.Parameters) == 0 {
			continue
		}
		result.WriteString("  Parameters:\n")
		paramNames := make([]string, 0, len(t.Parameters))
		for name := range t.Parameters {
			paramNames = append(paramNames, name)
		}
		sort.Strings(paramNames)
		for _, name := range paramNames {
			p := t.Parameters[name]
			req := ""
			if p.Required {
				req = " (required)"
			}
			result.WriteString(fmt.Sprintf("    - %s (%s): %s%s\n", p.Name, p.Type, p.Description, req))
		}
	}
	return result.String()
}

func (r *ToolRegistry) AsExecutor(guard *Guardrails) *PlanExecutor {
	exec := NewPlanExecutor(guard)
	for name, tool := range r.tools {
		if tool.Handler != nil {
			exec.RegisterHandler(name, tool.Handler)
		}
	}
	return exec
}

func RegisterGitTools(registry *ToolRegistry, gitRunner func(ctx context.Context, args ...string) (string, error)) {
	gitTool := func(action string, desc string, params map[string]ToolParam, fn func(ctx context.Context, args map[string]string) (string, error)) {
		registry.Register(Tool{
			Name:        action,
			Description: desc,
			Parameters:  params,
			Handler:     fn,
		})
	}

	gitTool("git.status", "Show repository status", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return gitRunner(ctx, "status", "--short", "--branch")
	})

	gitTool("git.add", "Stage files", map[string]ToolParam{
		"path": {Name: "path", Type: "string", Description: "Path to stage", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		path := args["path"]
		if path == "" {
			path = "."
		}
		return gitRunner(ctx, "add", path)
	})

	gitTool("git.commit", "Create a commit", map[string]ToolParam{
		"message": {Name: "message", Type: "string", Description: "Commit message", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return gitRunner(ctx, "commit", "-m", args["message"])
	})

	gitTool("git.push", "Push changes to remote", map[string]ToolParam{
		"remote": {Name: "remote", Type: "string", Description: "Remote name", Required: false},
		"branch": {Name: "branch", Type: "string", Description: "Branch name", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		cmdArgs := []string{"push"}
		if remote := args["remote"]; remote != "" {
			cmdArgs = append(cmdArgs, remote)
		}
		if branch := args["branch"]; branch != "" {
			cmdArgs = append(cmdArgs, branch)
		}
		return gitRunner(ctx, cmdArgs...)
	})

	gitTool("git.fetch", "Fetch remote updates", map[string]ToolParam{
		"remote": {Name: "remote", Type: "string", Description: "Remote name", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		cmdArgs := []string{"fetch"}
		if remote := args["remote"]; remote != "" {
			cmdArgs = append(cmdArgs, remote)
		}
		return gitRunner(ctx, cmdArgs...)
	})

	gitTool("git.pull", "Pull and integrate upstream changes", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return gitRunner(ctx, "pull")
	})

	gitTool("git.branch.create", "Create a branch", map[string]ToolParam{
		"name": {Name: "name", Type: "string", Description: "Branch name", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return gitRunner(ctx, "branch", args["name"])
	})

	gitTool("git.branch.delete", "Delete a branch", map[string]ToolParam{
		"name": {Name: "name", Type: "string", Description: "Branch name", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return gitRunner(ctx, "branch", "-d", args["name"])
	})

	gitTool("git.tag", "Create a tag", map[string]ToolParam{
		"name": {Name: "name", Type: "string", Description: "Tag name", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return gitRunner(ctx, "tag", args["name"])
	})

	gitTool("git.stash", "Create a stash entry", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return gitRunner(ctx, "stash")
	})

	gitTool("git.merge", "Merge a branch", map[string]ToolParam{
		"branch": {Name: "branch", Type: "string", Description: "Source branch", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return gitRunner(ctx, "merge", args["branch"])
	})

	gitTool("git.rebase", "Rebase onto another branch", map[string]ToolParam{
		"branch": {Name: "branch", Type: "string", Description: "Target branch", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return gitRunner(ctx, "rebase", args["branch"])
	})

	gitTool("git.gc", "Run git garbage collection", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return gitRunner(ctx, "gc")
	})

	gitTool("git.clean", "Remove untracked files", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return gitRunner(ctx, "clean", "-fd")
	})
}
