package gitops

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type IntegrityChecker struct {
	executor *GitExecutor
}

type FsckResult struct {
	Clean    bool
	Dangling []DanglingObject
	Missing  []string
	Corrupt  []string
	Warnings []string
}

type DanglingObject struct {
	Type string
	SHA  string
}

type ReflogEntry struct {
	SHA     string
	Action  string
	Message string
	Date    time.Time
}

func NewIntegrityChecker(executor *GitExecutor) *IntegrityChecker {
	return &IntegrityChecker{executor: executor}
}

func (ic *IntegrityChecker) Fsck(ctx context.Context, repoPath string, full bool) (*FsckResult, error) {
	args := []string{"fsck"}
	if full {
		args = append(args, "--full")
	} else {
		args = append(args, "--no-dangling")
	}

	result, err := ic.executor.Run(ctx, repoPath, args...)
	if err != nil {
		gerr, ok := err.(*GitError)
		if !ok {
			return nil, err
		}
		return ic.parseFsckOutput(gerr.Stderr, false), nil
	}
	return ic.parseFsckOutput(result.Stderr, true), nil
}

var (
	fsckDanglingRe = regexp.MustCompile(`dangling (blob|commit|tree|tag) ([a-f0-9]{40})`)
	fsckMissingRe  = regexp.MustCompile(`missing (blob|commit|tree) ([a-f0-9]{40})`)
	fsckCorruptRe  = regexp.MustCompile(`(?:error:|corrupt) .*?([a-f0-9]{40})`)
	fsckErrorRe    = regexp.MustCompile(`error: (.+)`)
)

func (ic *IntegrityChecker) parseFsckOutput(stderr string, clean bool) *FsckResult {
	r := &FsckResult{Clean: clean}

	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := fsckDanglingRe.FindStringSubmatch(line); len(m) == 3 {
			r.Dangling = append(r.Dangling, DanglingObject{Type: m[1], SHA: m[2]})
			r.Clean = false
		} else if m := fsckMissingRe.FindStringSubmatch(line); len(m) == 3 {
			r.Missing = append(r.Missing, m[2])
			r.Clean = false
		} else if m := fsckCorruptRe.FindStringSubmatch(line); len(m) >= 2 {
			r.Corrupt = append(r.Corrupt, m[len(m)-1])
			r.Clean = false
		} else if m := fsckErrorRe.FindStringSubmatch(line); len(m) == 2 {
			r.Warnings = append(r.Warnings, m[1])
			r.Clean = false
		}
	}
	return r
}

func (ic *IntegrityChecker) Reflog(ctx context.Context, repoPath, ref string, count int) ([]ReflogEntry, error) {
	args := []string{"reflog", "show", "--format=%H%x00%gs%x00%ai"}
	if ref != "" {
		args = append(args, ref)
	}
	if count > 0 {
		args = append(args, "-n", strconv.Itoa(count))
	}

	lines, err := ic.executor.RunLines(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}

	var entries []ReflogEntry
	for _, line := range lines {
		parts := strings.SplitN(line, "\x00", 3)
		if len(parts) < 2 {
			continue
		}
		e := ReflogEntry{SHA: strings.TrimSpace(parts[0])}
		gs := strings.TrimSpace(parts[1])
		if idx := strings.Index(gs, ": "); idx >= 0 {
			e.Action = strings.TrimSpace(gs[:idx])
			e.Message = strings.TrimSpace(gs[idx+2:])
		} else {
			e.Action = gs
		}
		if len(parts) >= 3 {
			if t, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(parts[2])); err == nil {
				e.Date = t
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (ic *IntegrityChecker) ReflogExpire(ctx context.Context, repoPath, expire string) error {
	args := []string{"reflog", "expire", "--expire=" + expire, "--all"}
	_, err := ic.executor.Run(ctx, repoPath, args...)
	return err
}

func (ic *IntegrityChecker) Prune(ctx context.Context, repoPath, expire string) error {
	args := []string{"prune"}
	if expire != "" {
		args = append(args, "--expire="+expire)
	}
	_, err := ic.executor.Run(ctx, repoPath, args...)
	return err
}

func (ic *IntegrityChecker) Repack(ctx context.Context, repoPath string, aggressive bool) error {
	args := []string{"repack", "-a", "-d"}
	if aggressive {
		args = append(args, "--depth=250", "--window=250")
	}
	_, err := ic.executor.Run(ctx, repoPath, args...)
	return err
}

func (ic *IntegrityChecker) PackRefs(ctx context.Context, repoPath string) error {
	_, err := ic.executor.Run(ctx, repoPath, "pack-refs", "--all")
	return err
}

func (ic *IntegrityChecker) Maintenance(ctx context.Context, repoPath, task string) error {
	_, err := ic.executor.Run(ctx, repoPath, "maintenance", "run", "--task="+task)
	return err
}

func (ic *IntegrityChecker) Archive(ctx context.Context, repoPath, ref, format, outputPath string) error {
	args := []string{"archive"}
	if format != "" {
		args = append(args, "--format="+format)
	}
	args = append(args, "-o", outputPath)
	if ref != "" {
		args = append(args, ref)
	} else {
		args = append(args, "HEAD")
	}
	_, err := ic.executor.Run(ctx, repoPath, args...)
	return err
}
