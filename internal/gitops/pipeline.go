package gitops

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// PipelineAction identifies the type of pipeline action.
type PipelineAction string

const (
	PipelineActionSync    PipelineAction = "sync"
	PipelineActionHygiene PipelineAction = "hygiene"
	PipelineActionPatch   PipelineAction = "patch"
	PipelineActionBranch  PipelineAction = "branch"
)

// RollbackType identifies how to rollback on failure.
type RollbackType string

const (
	RollbackDiscard      RollbackType = "discard"
	RollbackRevert       RollbackType = "revert"
	RollbackCompensation RollbackType = "compensation"
	RollbackHandoff      RollbackType = "handoff"
)

// GitPipeline orchestrates git change operations with locking, mirroring, and evidence collection.
type GitPipeline struct {
	executor   *GitExecutor
	mirrors    *MirrorManager
	worktrees  *WorktreeManager
	writerLock *WriterLock
	evidence   *EvidenceCollector
	inspector  *Inspector
}

// PipelineRequest describes a pipeline execution request.
type PipelineRequest struct {
	TaskID        string
	CorrelationID string
	Owner         string
	Repo          string
	TargetRef     string
	Action        PipelineAction
	Params        map[string]string
}

// PipelineResult holds the outcome of a pipeline execution.
type PipelineResult struct {
	Success  bool
	Evidence *ExecutionEvidence
	Rollback RollbackType
	Error    error
}

// NewGitPipeline creates a new GitPipeline with the given dependencies.
func NewGitPipeline(executor *GitExecutor, mirrors *MirrorManager, worktrees *WorktreeManager, writerLock *WriterLock, evidence *EvidenceCollector, inspector *Inspector) *GitPipeline {
	return &GitPipeline{
		executor:   executor,
		mirrors:    mirrors,
		worktrees:  worktrees,
		writerLock: writerLock,
		evidence:   evidence,
		inspector:  inspector,
	}
}

// Execute runs the pipeline: acquire lock, ensure mirror, optionally create worktree,
// collect state, dispatch action, collect post state, archive evidence, cleanup, release lock.
func (p *GitPipeline) Execute(ctx context.Context, req PipelineRequest) *PipelineResult {
	res := &PipelineResult{Rollback: RollbackDiscard}
	owner, repo, ref := req.Owner, req.Repo, req.TargetRef
	if ref == "" {
		ref = "main"
	}

	// 1. Acquire writer lock
	if p.writerLock != nil {
		if err := p.writerLock.Acquire(owner, repo, ref, req.TaskID); err != nil {
			res.Error = fmt.Errorf("acquire writer lock: %w", err)
			return res
		}
		defer func() {
			_ = p.writerLock.Release(owner, repo, ref, req.TaskID)
		}()
	}

	var repoPath string
	var worktreeDir string

	// 2. Ensure mirror or use repoPath from params
	if p.mirrors != nil && owner != "" && repo != "" {
		cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
		path, err := p.mirrors.EnsureMirror(ctx, cloneURL)
		if err != nil {
			res.Error = fmt.Errorf("ensure mirror: %w", err)
			return res
		}
		repoPath = path
	} else if req.Params != nil && req.Params["repoPath"] != "" {
		repoPath = req.Params["repoPath"]
	} else {
		res.Error = fmt.Errorf("no mirror and no repoPath in params")
		return res
	}

	// 3. (Optional) Create worktree for isolated execution
	if p.worktrees != nil && req.Params != nil && req.Params["useWorktree"] == "true" {
		wtDir := filepath.Join(filepath.Dir(repoPath), "gitdex-worktree-"+strings.ReplaceAll(ref, "/", "-"))
		wt, err := p.worktrees.Create(ctx, WorktreeConfig{
			RepoPath:    repoPath,
			Branch:      ref,
			WorktreeDir: wtDir,
		})
		if err != nil {
			res.Error = fmt.Errorf("create worktree: %w", err)
			return res
		}
		worktreeDir = wt.Config.WorktreeDir
		repoPath = worktreeDir
		defer func() {
			if worktreeDir != "" {
				if res.Success {
					_ = p.worktrees.Discard(ctx, worktreeDir)
				}
				// On failure, keep worktree for inspection
			}
		}()
	}

	start := time.Now()
	var diffBefore, diffAfter string

	// 4. Collect pre-execution state
	if p.inspector != nil {
		if insp, err := p.inspector.Inspect(ctx, repoPath); err == nil {
			diffBefore = fmt.Sprintf("branch=%s remote=%s ahead=%d behind=%d",
				insp.LocalBranch, insp.RemoteBranch, insp.Ahead, insp.Behind)
		}
	}
	if p.worktrees != nil && worktreeDir != "" {
		if diff, err := p.worktrees.Diff(ctx, worktreeDir); err == nil && diff != "" {
			diffBefore = diffBefore + "\n--- diff ---\n" + diff
		}
	}

	// 5. Execute action (dispatch sync/hygiene/patch/branch)
	if err := p.executeAction(ctx, repoPath, &req); err != nil {
		res.Error = err
		res.Rollback = RollbackDiscard
		res.Evidence = p.buildEvidence(req, start, diffBefore, "", err, nil)
		_ = p.evidence.Collect(res.Evidence)
		return res
	}

	// 6. Collect post-execution state
	if p.inspector != nil {
		if insp, err := p.inspector.Inspect(ctx, repoPath); err == nil {
			diffAfter = fmt.Sprintf("branch=%s remote=%s ahead=%d behind=%d",
				insp.LocalBranch, insp.RemoteBranch, insp.Ahead, insp.Behind)
		}
	}
	if p.worktrees != nil && worktreeDir != "" {
		if diff, err := p.worktrees.Diff(ctx, worktreeDir); err == nil && diff != "" {
			diffAfter = diffAfter + "\n--- diff ---\n" + diff
		}
	}

	res.Success = true
	res.Evidence = p.buildEvidence(req, start, diffBefore, diffAfter, nil, nil)
	res.Evidence.DiffBefore = diffBefore
	res.Evidence.DiffAfter = diffAfter

	// 7. Archive evidence
	_ = p.evidence.Collect(res.Evidence)

	return res
}

func (p *GitPipeline) executeAction(ctx context.Context, repoPath string, req *PipelineRequest) error {
	switch req.Action {
	case PipelineActionSync:
		return p.actionSync(ctx, repoPath, req)
	case PipelineActionHygiene:
		return p.actionHygiene(ctx, repoPath, req)
	case PipelineActionPatch:
		return p.actionPatch(ctx, repoPath, req)
	case PipelineActionBranch:
		return p.actionBranch(ctx, repoPath, req)
	default:
		return fmt.Errorf("unknown pipeline action: %s", req.Action)
	}
}

func (p *GitPipeline) actionSync(ctx context.Context, repoPath string, req *PipelineRequest) error {
	// Sync: fetch and merge/rebase to bring branch up to date
	_, err := p.executor.Run(ctx, repoPath, "fetch", "origin", req.TargetRef)
	if err != nil {
		return err
	}
	_, err = p.executor.Run(ctx, repoPath, "merge", "--ff-only", "origin/"+req.TargetRef)
	return err
}

func (p *GitPipeline) actionHygiene(ctx context.Context, repoPath string, _ *PipelineRequest) error {
	// Hygiene: gc, prune, repack
	_, err := p.executor.Run(ctx, repoPath, "gc", "--auto")
	return err
}

func (p *GitPipeline) actionPatch(ctx context.Context, repoPath string, req *PipelineRequest) error {
	// Patch: apply patch from params if provided
	if req.Params != nil {
		if patchPath := req.Params["patchPath"]; patchPath != "" {
			_, err := p.executor.Run(ctx, repoPath, "apply", patchPath)
			return err
		}
	}
	return nil
}

func (p *GitPipeline) actionBranch(ctx context.Context, repoPath string, req *PipelineRequest) error {
	// Branch: create/checkout branch from params
	if req.Params == nil {
		return nil
	}
	newBranch := req.Params["newBranch"]
	if newBranch == "" {
		return nil
	}
	startPoint := req.Params["startPoint"]
	if startPoint == "" {
		startPoint = req.TargetRef
	}
	_, err := p.executor.Run(ctx, repoPath, "checkout", "-b", newBranch, startPoint)
	return err
}

func (p *GitPipeline) buildEvidence(req PipelineRequest, start time.Time, diffBefore, diffAfter string, err error, commands []GitCommandRecord) *ExecutionEvidence {
	ev := &ExecutionEvidence{
		TaskID:        req.TaskID,
		CorrelationID: req.CorrelationID,
		Action:        string(req.Action),
		RepoPath:      req.Owner + "/" + req.Repo,
		Timestamp:     start.UTC(),
		Duration:      time.Since(start),
		DiffBefore:    diffBefore,
		DiffAfter:     diffAfter,
		GitCommands:   commands,
	}
	if err != nil {
		ev.Result = "failed"
		ev.ErrorDetail = err.Error()
	} else {
		ev.Result = "success"
	}
	return ev
}
