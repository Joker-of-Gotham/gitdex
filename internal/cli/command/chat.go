package command

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/app/chat"
	"github.com/your-org/gitdex/internal/app/session"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/cli/input"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/llm/adapter"
	"github.com/your-org/gitdex/internal/platform/config"
)

func newChatCommand(flags *runtimeOptions, appFn func() bootstrap.App, sessionCtx **session.TaskContext, provider *adapter.Provider) *cobra.Command {
	var interactive bool
	var execute bool
	var repoFlag string
	var pathFlag string
	var autoThreshold string
	var approvalThreshold string

	cmd := &cobra.Command{
		Use:   "chat [message]",
		Short: "Start a natural language conversation with Gitdex",
		Long: `Start a natural language conversation with Gitdex.

In single-message mode, pass a message as an argument:
  gitdex chat "What commands are available?"

In interactive mode, enter a REPL session:
  gitdex chat --interactive

Inside interactive mode:
  - Type natural language to chat with Gitdex
  - Type !<intent> to plan and execute a repository action from natural language
  - Type a known command name (e.g. "doctor") to run it
  - Type "exit" or "quit" to leave
  - Press Ctrl+C to abort`,
		Example: `  gitdex chat "What can you help me with?"
  gitdex chat --interactive
  gitdex chat --execute "add a release checklist and commit it"
  gitdex chat "Summarize my recent doctor results" --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			tc := getOrCreateSession(sessionCtx, app.RepoRoot, flags)
			resolvedProvider, err := resolveProvider(provider, app.Config.LLM)
			if err != nil {
				return err
			}
			svc := chat.NewService(resolvedProvider)

			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			if interactive {
				return runInteractiveChat(cmd, svc, tc, format, cmd.InOrStdin(), cmd.OutOrStdout(), app, execute, repoFlag, pathFlag, autoThreshold, approvalThreshold)
			}

			if len(args) == 0 {
				return fmt.Errorf("provide a message argument or use --interactive for REPL mode")
			}

			message := strings.Join(args, " ")
			if execute {
				return runSingleChatIntent(cmd, tc, format, app, message, repoFlag, pathFlag, true, autoThreshold, approvalThreshold)
			}
			return runSingleChat(cmd, svc, tc, format, message)
		},
	}

	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Start interactive REPL chat session")
	cmd.Flags().BoolVar(&execute, "execute", false, "Interpret the message as an executable natural-language intent")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "Repository owner/repo for remote-aware execution context")
	cmd.Flags().StringVar(&pathFlag, "path", "", "Explicit local clone path for file and Git execution")
	cmd.Flags().StringVar(&autoThreshold, "auto-threshold", autonomy.RiskHigh.String(), "Auto-execution threshold: low, medium, high, critical")
	cmd.Flags().StringVar(&approvalThreshold, "approval-threshold", autonomy.RiskCritical.String(), "Threshold above which plans stay pending")

	return cmd
}

func runSingleChat(cmd *cobra.Command, svc *chat.Service, tc *session.TaskContext, format, message string) error {
	result, err := svc.Chat(cmd.Context(), tc, message)
	if err != nil {
		return err
	}

	if clioutput.IsStructured(format) {
		return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
	}

	_, writeErr := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", result.Content)
	return writeErr
}

func runSingleChatIntent(cmd *cobra.Command, tc *session.TaskContext, format string, app bootstrap.App, message, repoFlag, pathFlag string, execute bool, autoThreshold, approvalThreshold string) error {
	tc.AddChatMessage(session.ChatMessage{Role: "user", Content: message})

	repoRoot := firstNonEmpty(pathFlag, app.RepoRoot, app.Config.Paths.RepositoryRoot, tc.GetRepoPath())
	owner, repoName := parseRepoFlag(repoFlag, repoRoot)
	if owner == "" || repoName == "" {
		owner, repoName = resolveOwnerRepo("", "", repoRoot)
	}
	repoRoot = selectRepoRootForRemote(app, repoRoot, owner, repoName)

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := runAutonomyCycle(ctx, cmd, app, autonomyRunRequest{
		RepoRoot:          repoRoot,
		Owner:             owner,
		Repo:              repoName,
		Intent:            message,
		Execute:           execute,
		AutoThreshold:     autonomy.ParseRiskLevel(autoThreshold),
		ApprovalThreshold: autonomy.ParseRiskLevel(approvalThreshold),
	})
	if err != nil {
		return err
	}

	tc.AddChatMessage(session.ChatMessage{
		Role:    "assistant",
		Content: fmt.Sprintf("Autonomy %s cycle %s planned %d action(s)", result.Mode, result.Report.CycleID, len(result.Plans)),
	})

	if clioutput.IsStructured(format) {
		return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
	}

	var buf bytes.Buffer
	if err := renderAutonomyRunResult(&buf, result); err != nil {
		return err
	}
	_, writeErr := fmt.Fprintln(cmd.OutOrStdout(), strings.TrimRight(buf.String(), "\n"))
	return writeErr
}

func runInteractiveChat(cmd *cobra.Command, svc *chat.Service, tc *session.TaskContext, format string, in io.Reader, out io.Writer, app bootstrap.App, executeByDefault bool, repoFlag, pathFlag, autoThreshold, approvalThreshold string) error {
	parser := input.NewParser(cmd.Root())
	scanner := bufio.NewScanner(in)

	_, _ = fmt.Fprintln(out, "Gitdex interactive chat (type 'exit' to quit)")
	_, _ = fmt.Fprintln(out, "---")

	for {
		_, _ = fmt.Fprint(out, "you> ")
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}
		if trimmed == "exit" || trimmed == "quit" {
			_, _ = fmt.Fprintln(out, "Goodbye.")
			return nil
		}
		if strings.HasPrefix(trimmed, "!") {
			intent := strings.TrimSpace(strings.TrimPrefix(trimmed, "!"))
			if intent == "" {
				_, _ = fmt.Fprintln(out, "[usage: !<intent>]")
				continue
			}
			if err := runSingleChatIntent(cmd, tc, format, app, intent, repoFlag, pathFlag, true, autoThreshold, approvalThreshold); err != nil {
				_, _ = fmt.Fprintf(out, "[execute error: %v]\n", err)
			}
			continue
		}
		if executeByDefault {
			if err := runSingleChatIntent(cmd, tc, format, app, trimmed, repoFlag, pathFlag, true, autoThreshold, approvalThreshold); err != nil {
				_, _ = fmt.Fprintf(out, "[execute error: %v]\n", err)
			}
			continue
		}

		classified := parser.Classify(trimmed)
		if classified.Type == input.InputCommand {
			_, _ = fmt.Fprintf(out, "[command detected: %s — run it directly via `gitdex %s`]\n",
				classified.Command, strings.Join(strings.Fields(trimmed), " "))
			tc.AddChatMessage(session.ChatMessage{
				Role:    "system",
				Content: fmt.Sprintf("Operator attempted to run command: %s", classified.Command),
			})
			continue
		}

		result, err := svc.Chat(cmd.Context(), tc, trimmed)
		if err != nil {
			_, _ = fmt.Fprintf(out, "[error: %v]\n", err)
			continue
		}

		if clioutput.IsStructured(format) {
			if writeErr := clioutput.WriteValue(out, format, result); writeErr != nil {
				_, _ = fmt.Fprintf(out, "[output error: %v]\n", writeErr)
			}
		} else {
			_, _ = fmt.Fprintf(out, "gitdex> %s\n", result.Content)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	return nil
}

func getOrCreateSession(sessionCtx **session.TaskContext, repoRoot string, flags *runtimeOptions) *session.TaskContext {
	if *sessionCtx != nil {
		return *sessionCtx
	}
	tc := session.NewTaskContext(repoRoot, flags.profile)
	*sessionCtx = tc
	return tc
}

func resolveProvider(p *adapter.Provider, llmCfg config.LLMConfig) (adapter.Provider, error) {
	if p != nil && *p != nil {
		return *p, nil
	}

	providerName := firstNonEmpty(os.Getenv("GITDEX_LLM_PROVIDER"), llmCfg.Provider)
	apiKey := firstNonEmpty(os.Getenv("GITDEX_LLM_API_KEY"), llmCfg.APIKey)
	endpoint := firstNonEmpty(os.Getenv("GITDEX_LLM_ENDPOINT"), llmCfg.Endpoint)
	model := firstNonEmpty(os.Getenv("GITDEX_LLM_MODEL"), llmCfg.Model)
	if providerName == "" {
		providerName = "openai"
	}

	if !strings.EqualFold(providerName, "ollama") && strings.TrimSpace(apiKey) == "" {
		return &adapter.MockProvider{}, nil
	}

	provider, err := adapter.NewProviderFromConfig(providerName, model, apiKey, endpoint)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
