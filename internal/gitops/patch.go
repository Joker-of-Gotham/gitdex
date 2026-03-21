package gitops

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type DiffOptions struct {
	Cached           bool
	NameOnly         bool
	Stat             bool
	NumStat          bool
	Paths            []string
	ContextLines     int
	IgnoreWhitespace bool
}

type DiffEntry struct {
	Path      string
	Status    string
	Additions int
	Deletions int
	OldPath   string
}

type PatchResult struct {
	Files        []string
	TotalAdded   int
	TotalRemoved int
	PatchPath    string
}

type PatchManager struct {
	executor *GitExecutor
}

func NewPatchManager(executor *GitExecutor) *PatchManager {
	return &PatchManager{executor: executor}
}

func (pm *PatchManager) Diff(ctx context.Context, repoPath string, opts *DiffOptions) (string, error) {
	args := pm.buildDiffArgs(opts)
	result, err := pm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func (pm *PatchManager) DiffBetween(ctx context.Context, repoPath string, from string, to string, opts *DiffOptions) (string, error) {
	args := pm.buildDiffArgs(opts)
	args = append(args, from, to)
	if opts != nil && len(opts.Paths) > 0 {
		args = append(args, "--")
		args = append(args, opts.Paths...)
	}
	result, err := pm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func (pm *PatchManager) buildDiffArgs(opts *DiffOptions) []string {
	args := []string{"diff"}
	if opts != nil {
		if opts.Cached {
			args = append(args, "--cached")
		}
		if opts.NameOnly {
			args = append(args, "--name-only")
		}
		if opts.Stat {
			args = append(args, "--stat")
		}
		if opts.NumStat {
			args = append(args, "--numstat")
		}
		if opts.ContextLines > 0 {
			args = append(args, "-U", strconv.Itoa(opts.ContextLines))
		}
		if opts.IgnoreWhitespace {
			args = append(args, "-w")
		}
	}
	return args
}

func (pm *PatchManager) DiffStat(ctx context.Context, repoPath string, from string, to string) ([]DiffEntry, error) {
	args := []string{"diff", "--numstat"}
	if from != "" && to != "" {
		args = append(args, from, to)
	} else if from != "" {
		args = append(args, from)
	}
	result, err := pm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	return pm.parseNumStat(result.Stdout), nil
}

func (pm *PatchManager) DiffNameOnly(ctx context.Context, repoPath string, cached bool) ([]string, error) {
	args := []string{"diff", "--name-only"}
	if cached {
		args = append(args, "--cached")
	}
	return pm.executor.RunLines(ctx, repoPath, args...)
}

func (pm *PatchManager) DiffNumstat(ctx context.Context, repoPath string, from string, to string) ([]DiffEntry, error) {
	args := []string{"diff", "--numstat"}
	if from != "" && to != "" {
		args = append(args, from, to)
	} else if from != "" {
		args = append(args, from)
	}
	result, err := pm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	return pm.parseNumStat(result.Stdout), nil
}

func (pm *PatchManager) parseNumStat(stdout string) []DiffEntry {
	var entries []DiffEntry
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		e := DiffEntry{}
		if len(parts) >= 1 && parts[0] != "-" {
			if add, err := strconv.Atoi(parts[0]); err == nil {
				e.Additions = add
			}
		}
		if len(parts) >= 2 && parts[1] != "-" {
			if del, err := strconv.Atoi(parts[1]); err == nil {
				e.Deletions = del
			}
		}
		if len(parts) >= 3 {
			e.Path = strings.TrimSpace(parts[2])
		}
		entries = append(entries, e)
	}
	return entries
}

func (pm *PatchManager) CreatePatch(ctx context.Context, repoPath string, outPath string, from string, to string) (*PatchResult, error) {
	args := []string{"diff"}
	if from != "" && to != "" {
		args = append(args, from, to)
	} else if from != "" {
		args = append(args, from)
	}
	result, err := pm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	if outPath != "" {
		if err := os.WriteFile(outPath, []byte(result.Stdout), 0644); err != nil {
			return nil, err
		}
	}
	statArgs := []string{"diff", "--numstat"}
	if from != "" && to != "" {
		statArgs = append(statArgs, from, to)
	} else if from != "" {
		statArgs = append(statArgs, from)
	}
	statResult, _ := pm.executor.Run(ctx, repoPath, statArgs...)
	entries := []DiffEntry{}
	if statResult != nil {
		entries = pm.parseNumStat(statResult.Stdout)
	}
	files := pm.parseFileListFromDiff(result.Stdout)
	pr := &PatchResult{PatchPath: outPath, Files: files}
	for _, e := range entries {
		pr.TotalAdded += e.Additions
		pr.TotalRemoved += e.Deletions
	}
	return pr, nil
}

func (pm *PatchManager) parseFileListFromDiff(patch string) []string {
	var files []string
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				path := strings.TrimPrefix(parts[2], "a/")
				files = append(files, path)
			}
		}
	}
	return files
}

func (pm *PatchManager) CreatePatchFiles(ctx context.Context, repoPath string, outDir string, from string, to string) ([]string, error) {
	args := []string{"format-patch", "-o", outDir}
	if from != "" && to != "" {
		args = append(args, from+".."+to)
	} else if from != "" {
		args = append(args, from)
	}
	lines, err := pm.executor.RunLines(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, filepath.Join(outDir, line))
		}
	}
	return paths, nil
}

func (pm *PatchManager) ApplyPatch(ctx context.Context, repoPath string, patchPath string, check bool) error {
	args := []string{"apply"}
	if check {
		args = append(args, "--check")
	}
	args = append(args, patchPath)
	_, err := pm.executor.Run(ctx, repoPath, args...)
	return err
}

// ApplyPatchReverse runs git apply -R <patchPath>.
func (pm *PatchManager) ApplyPatchReverse(ctx context.Context, repoPath string, patchPath string) error {
	args := []string{"apply", "-R", patchPath}
	_, err := pm.executor.Run(ctx, repoPath, args...)
	return err
}

func (pm *PatchManager) ApplyPatchFromStdin(ctx context.Context, repoPath string, stdin io.Reader, check bool) error {
	args := []string{"apply"}
	if check {
		args = append(args, "--check")
	}
	args = append(args, "-")
	_, err := pm.executor.RunWithInput(ctx, repoPath, stdin, args...)
	return err
}

// ApplyPatchCachedFromStdin runs git apply --cached (optionally -R) to stage or unstage index hunks.
func (pm *PatchManager) ApplyPatchCachedFromStdin(ctx context.Context, repoPath string, stdin io.Reader, reverse bool) error {
	args := []string{"apply", "--cached"}
	if reverse {
		args = append(args, "-R")
	}
	args = append(args, "-")
	_, err := pm.executor.RunWithInput(ctx, repoPath, stdin, args...)
	return err
}

func (pm *PatchManager) RangeDiff(ctx context.Context, repoPath string, base string, from string, to string) (string, error) {
	args := []string{"range-diff", base + ".." + from, base + ".." + to}
	result, err := pm.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}
