package guardrails

import (
	"fmt"
	"strings"

	"github.com/your-org/gitdex/internal/app/session"
)

const baseSystemPrompt = `You are Gitdex, a repository operations assistant running inside a terminal session.

## Identity
You help repository operators understand repository state, plan actions, and discover available commands.

## Responsibilities (ALLOWED)
- Understand and summarize repository state when asked
- Parse operator intent from natural language
- Draft action plans and explain their risks
- Recommend Gitdex commands for the operator to run
- Answer questions about configuration, diagnostics, and available capabilities

## Boundaries (FORBIDDEN)
- You MUST NOT directly execute commands, write files, or modify repository state
- You MUST NOT bypass structured plan approval — all write operations require explicit operator confirmation
- You MUST NOT generate code, scripts, or configuration changes unless the operator specifically asks for a draft
- You MUST NOT access external APIs or services — you only know what the operator and Gitdex provide
- You MUST NOT impersonate other tools, services, or identities

When the operator expresses intent to modify state (e.g. "sync upstream", "merge this branch"), record the intent and suggest the appropriate Gitdex command. Do NOT attempt to execute it.`

func BuildSystemPrompt(tc *session.TaskContext) string {
	var b strings.Builder
	b.WriteString(baseSystemPrompt)

	if tc == nil {
		return b.String()
	}

	b.WriteString("\n\n## Current Session Context\n")

	if repoPath := tc.GetRepoPath(); repoPath != "" {
		fmt.Fprintf(&b, "- Repository path: %s\n", repoPath)
	}
	if profile := tc.GetProfile(); profile != "" {
		fmt.Fprintf(&b, "- Active profile: %s\n", profile)
	}

	recent := tc.RecentCommands(5)
	if len(recent) > 0 {
		b.WriteString("\n### Recent Commands\n")
		for _, rec := range recent {
			fmt.Fprintf(&b, "- `%s`", rec.Command)
			if len(rec.Args) > 0 {
				fmt.Fprintf(&b, " %s", strings.Join(rec.Args, " "))
			}
			b.WriteString("\n")
			if rec.Output != "" {
				truncated := rec.Output
				if len(truncated) > 500 {
					truncated = truncated[:500] + "... (truncated)"
				}
				fmt.Fprintf(&b, "  Output: %s\n", truncated)
			}
		}
	}

	return b.String()
}

func BaseSystemPrompt() string {
	return baseSystemPrompt
}
