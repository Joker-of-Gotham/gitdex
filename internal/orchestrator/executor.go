package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/your-org/gitdex/internal/gitops"
	"github.com/your-org/gitdex/internal/planning"
)

const TaskStatusCreated TaskStatus = "created"

type RepoRootResolver func(ctx context.Context, plan *planning.Plan) (string, error)
type StepExecutor func(ctx context.Context, plan *planning.Plan, repoRoot string, step *StepResult) (string, error)

type ExecutorOptions struct {
	RepoRoot         string
	WorkspaceRoots   []string
	GitExecutor      *gitops.GitExecutor
	RepoRootResolver RepoRootResolver
	StepExecutor     StepExecutor
}

type Executor struct {
	taskStore TaskStore
	planStore planning.PlanStore
	mu        sync.Mutex
	running   map[string]bool

	repoRoot         string
	workspaceRoots   []string
	gitExecutor      *gitops.GitExecutor
	repoRootResolver RepoRootResolver
	stepExecutor     StepExecutor
}

func NewExecutor(taskStore TaskStore, planStore planning.PlanStore) *Executor {
	return NewExecutorWithOptions(taskStore, planStore, ExecutorOptions{})
}

func NewExecutorWithOptions(taskStore TaskStore, planStore planning.PlanStore, opts ExecutorOptions) *Executor {
	gitExec := opts.GitExecutor
	if gitExec == nil {
		gitExec = gitops.NewGitExecutor()
	}
	return &Executor{
		taskStore:        taskStore,
		planStore:        planStore,
		running:          make(map[string]bool),
		repoRoot:         opts.RepoRoot,
		workspaceRoots:   opts.WorkspaceRoots,
		gitExecutor:      gitExec,
		repoRootResolver: opts.RepoRootResolver,
		stepExecutor:     opts.StepExecutor,
	}
}

func (e *Executor) StartFromPlan(ctx context.Context, planID string) (*Task, error) {
	plan, err := e.planStore.Get(planID)
	if err != nil {
		return nil, fmt.Errorf("cannot find plan: %w", err)
	}
	if plan.Status != planning.PlanApproved {
		return nil, fmt.Errorf("plan %q is not approved (status: %s); only approved plans can be executed", planID, plan.Status)
	}
	if len(plan.Steps) == 0 {
		return nil, fmt.Errorf("plan %q has no steps to execute", planID)
	}

	now := time.Now().UTC()
	task := &Task{
		TaskID:        GenerateTaskID(),
		CorrelationID: GenerateCorrelationID(),
		PlanID:        planID,
		Status:        TaskQueued,
		CurrentStep:   0,
		Steps:         make([]StepResult, len(plan.Steps)),
		UpdatedAt:     now,
	}

	for i, s := range plan.Steps {
		task.Steps[i] = StepResult{
			Sequence:    s.Sequence,
			Action:      s.Action,
			Target:      s.Target,
			Description: s.Description,
			Status:      StepPending,
		}
	}

	if err := e.taskStore.SaveTask(task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	e.recordEvent(task.TaskID, TaskStatusCreated, TaskQueued, 0, "task created from plan "+planID)

	if err := e.planStore.UpdateStatus(planID, planning.PlanExecuting); err != nil {
		_ = e.taskStore.UpdateTask(&Task{TaskID: task.TaskID, Status: TaskCancelled, UpdatedAt: time.Now().UTC()})
		return nil, fmt.Errorf("failed to update plan status: %w", err)
	}

	return task, nil
}

func (e *Executor) Execute(ctx context.Context, task *Task) error {
	if task.Status != TaskQueued {
		return fmt.Errorf("task %q is not queued (status: %s)", task.TaskID, task.Status)
	}

	plan, err := e.planStore.Get(task.PlanID)
	if err != nil {
		return fmt.Errorf("cannot load plan %q: %w", task.PlanID, err)
	}

	repoRoot, err := e.resolveRepoRoot(ctx, plan)
	if err != nil {
		return fmt.Errorf("resolve execution repository: %w", err)
	}

	e.mu.Lock()
	if e.running[task.TaskID] {
		e.mu.Unlock()
		return fmt.Errorf("task %q is already being executed", task.TaskID)
	}
	e.running[task.TaskID] = true
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.running, task.TaskID)
		e.mu.Unlock()
	}()

	now := time.Now().UTC()
	task.Status = TaskExecuting
	task.StartedAt = &now
	if err := e.taskStore.UpdateTask(task); err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}
	e.recordEvent(task.TaskID, TaskQueued, TaskExecuting, 0, "execution started")

	for i := range task.Steps {
		if err := ctx.Err(); err != nil {
			task.Status = TaskCancelled
			e.persistBestEffort(task)
			e.recordEvent(task.TaskID, TaskExecuting, TaskCancelled, task.Steps[i].Sequence, "cancelled by context")
			return err
		}

		task.CurrentStep = task.Steps[i].Sequence
		stepStart := time.Now().UTC()
		task.Steps[i].Status = StepRunning
		task.Steps[i].StartedAt = &stepStart
		e.persistBestEffort(task)
		e.recordEvent(task.TaskID, TaskExecuting, TaskExecuting, task.Steps[i].Sequence,
			fmt.Sprintf("step %d started: %s", task.Steps[i].Sequence, task.Steps[i].Action))

		output, err := e.executeStep(ctx, plan, repoRoot, &task.Steps[i])
		stepEnd := time.Now().UTC()
		task.Steps[i].FinishedAt = &stepEnd

		if err != nil {
			task.Steps[i].Status = StepFailed
			task.Steps[i].ErrorMessage = err.Error()

			for j := i + 1; j < len(task.Steps); j++ {
				task.Steps[j].Status = StepSkipped
			}
			task.Status = TaskFailedHandoffPending
			finishTime := time.Now().UTC()
			task.FinishedAt = &finishTime
			e.persistBestEffort(task)
			e.recordEvent(task.TaskID, TaskExecuting, TaskFailedHandoffPending, task.Steps[i].Sequence,
				fmt.Sprintf("step %d failed: %s", task.Steps[i].Sequence, err.Error()))
			return fmt.Errorf("step %d failed: %w", task.Steps[i].Sequence, err)
		}

		task.Steps[i].Status = StepSucceeded
		task.Steps[i].Output = output
		e.persistBestEffort(task)
		e.recordEvent(task.TaskID, TaskExecuting, TaskExecuting, task.Steps[i].Sequence,
			fmt.Sprintf("step %d succeeded", task.Steps[i].Sequence))
	}

	task.Status = TaskReconciling
	e.persistBestEffort(task)
	e.recordEvent(task.TaskID, TaskExecuting, TaskReconciling, 0, "all steps completed, reconciling")

	task.Status = TaskSucceeded
	finishTime := time.Now().UTC()
	task.FinishedAt = &finishTime
	e.persistBestEffort(task)
	e.recordEvent(task.TaskID, TaskReconciling, TaskSucceeded, 0, "task succeeded")

	_ = e.planStore.UpdateStatus(task.PlanID, planning.PlanCompleted)

	return nil
}

func (e *Executor) executeStep(ctx context.Context, plan *planning.Plan, repoRoot string, step *StepResult) (string, error) {
	if e.stepExecutor != nil {
		return e.stepExecutor(ctx, plan, repoRoot, step)
	}
	return e.defaultExecuteStep(ctx, plan, repoRoot, step)
}

func (e *Executor) persistBestEffort(task *Task) {
	if err := e.taskStore.UpdateTask(task); err != nil {
		e.recordEvent(task.TaskID, task.Status, task.Status, task.CurrentStep,
			fmt.Sprintf("WARNING: failed to persist task state: %s", err.Error()))
	}
}

func (e *Executor) recordEvent(taskID string, from, to TaskStatus, stepSeq int, msg string) {
	_ = e.taskStore.AppendEvent(&TaskEvent{
		TaskID:       taskID,
		FromStatus:   from,
		ToStatus:     to,
		StepSequence: stepSeq,
		Message:      msg,
		Timestamp:    time.Now().UTC(),
	})
}
