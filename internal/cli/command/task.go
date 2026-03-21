package command

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/orchestrator"
)

func newTaskGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage and inspect governed task execution",
	}
	cmd.AddCommand(newTaskStartCommand(flags, appFn))
	cmd.AddCommand(newTaskStatusCommand(flags, appFn))
	cmd.AddCommand(newTaskListCommand(flags, appFn))
	return cmd
}

func newTaskStartCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "start [plan_id]",
		Short: "Start executing an approved plan as a tracked task",
		Long:  "Create a task from an approved plan and execute it with lifecycle tracking. Plans are in-memory only.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			taskStore := app.StorageProvider.TaskStore()
			planStore := app.StorageProvider.PlanStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			planID := args[0]

			exec := orchestrator.NewExecutorWithOptions(taskStore, planStore, orchestrator.ExecutorOptions{
				RepoRoot:       app.RepoRoot,
				WorkspaceRoots: app.Config.Git.WorkspaceRoots,
			})
			task, err := exec.StartFromPlan(context.Background(), planID)
			if err != nil {
				return err
			}

			execErr := exec.Execute(context.Background(), task)

			latest, _ := taskStore.GetTask(task.TaskID)
			if latest != nil {
				task = latest
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, task)
			}

			if err := renderTaskStatus(cmd.OutOrStdout(), task); err != nil {
				return err
			}
			if execErr != nil {
				return fmt.Errorf("task execution failed: %w", execErr)
			}
			return nil
		},
	}
}

func newTaskStatusCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "status [task_id]",
		Short: "Show current status and step progress of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			taskStore := app.StorageProvider.TaskStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			task, err := taskStore.GetTask(args[0])
			if err != nil {
				return fmt.Errorf("task not found (tasks are stored in memory for this session only): %w", err)
			}

			events, _ := taskStore.GetEvents(args[0])

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{
					"task":   task,
					"events": events,
				})
			}
			return renderTaskStatusWithEvents(cmd.OutOrStdout(), task, events)
		},
	}
}

func newTaskListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tracked tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			taskStore := app.StorageProvider.TaskStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			tasks, err := taskStore.ListTasks()
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, tasks)
			}
			return renderTaskList(cmd.OutOrStdout(), tasks)
		},
	}
}

func renderTaskStatus(out io.Writer, t *orchestrator.Task) error {
	_, _ = fmt.Fprintf(out, "Task: %s\n", t.TaskID)
	_, _ = fmt.Fprintf(out, "Correlation: %s\n", t.CorrelationID)
	_, _ = fmt.Fprintf(out, "Plan: %s\n", t.PlanID)
	_, _ = fmt.Fprintf(out, "Status: %s\n", t.Status)

	if t.StartedAt != nil {
		elapsed := time.Since(*t.StartedAt)
		_, _ = fmt.Fprintf(out, "Running for: %s\n", elapsed.Truncate(time.Second))

		if elapsed > 3*time.Second {
			_, _ = fmt.Fprintf(out, "Stage: step %d of %d\n", t.CurrentStep, len(t.Steps))
		}
		if elapsed > 10*time.Second && !t.Status.IsTerminal() {
			_, _ = fmt.Fprintf(out, "Note: safe to leave — task will continue in background\n")
		}
	}

	_, _ = fmt.Fprintf(out, "\nSteps:\n")
	for _, s := range t.Steps {
		icon := statusIcon(s.Status)
		_, _ = fmt.Fprintf(out, "  %s %d. %s → %s [%s]\n", icon, s.Sequence, s.Action, s.Target, s.Status)
		if s.ErrorMessage != "" {
			_, _ = fmt.Fprintf(out, "     Error: %s\n", s.ErrorMessage)
		}
		if s.Output != "" {
			_, _ = fmt.Fprintf(out, "     Output: %s\n", s.Output)
		}
	}

	return nil
}

func renderTaskStatusWithEvents(out io.Writer, t *orchestrator.Task, events []*orchestrator.TaskEvent) error {
	if err := renderTaskStatus(out, t); err != nil {
		return err
	}

	if len(events) > 0 {
		_, _ = fmt.Fprintf(out, "\n── Event Log ──\n")
		for _, e := range events {
			_, _ = fmt.Fprintf(out, "  %s | %s → %s | %s\n",
				e.Timestamp.Format(time.RFC3339), e.FromStatus, e.ToStatus, e.Message)
		}
	}

	return nil
}

func renderTaskList(out io.Writer, tasks []*orchestrator.Task) error {
	if len(tasks) == 0 {
		_, _ = fmt.Fprintln(out, "No tasks found.")
		return nil
	}

	_, _ = fmt.Fprintf(out, "%-20s %-15s %-20s %s\n", "Task ID", "Status", "Plan ID", "Step")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 75))
	for _, t := range tasks {
		stepInfo := fmt.Sprintf("%d/%d", t.CurrentStep, len(t.Steps))
		_, _ = fmt.Fprintf(out, "%-20s %-15s %-20s %s\n", t.TaskID, t.Status, t.PlanID, stepInfo)
	}
	return nil
}

func statusIcon(s orchestrator.StepStatus) string {
	switch s {
	case orchestrator.StepSucceeded:
		return "+"
	case orchestrator.StepFailed:
		return "x"
	case orchestrator.StepRunning:
		return ">"
	case orchestrator.StepSkipped:
		return "-"
	default:
		return "."
	}
}
