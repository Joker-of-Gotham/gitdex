package git

import (
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

type LocalGitState struct {
	Branch        string   `json:"branch" yaml:"branch"`
	HeadSHA       string   `json:"head_sha" yaml:"head_sha"`
	IsDetached    bool     `json:"is_detached" yaml:"is_detached"`
	IsClean       bool     `json:"is_clean" yaml:"is_clean"`
	StagedCount   int      `json:"staged_count" yaml:"staged_count"`
	DirtyCount    int      `json:"dirty_count" yaml:"dirty_count"`
	Ahead         int      `json:"ahead" yaml:"ahead"`
	Behind        int      `json:"behind" yaml:"behind"`
	Remotes       []string `json:"remotes" yaml:"remotes"`
	DefaultRemote string   `json:"default_remote,omitempty" yaml:"default_remote,omitempty"`
}

func ReadLocalState(repoPath string) (*LocalGitState, error) {
	r, err := gogit.PlainOpenWithOptions(repoPath, &gogit.PlainOpenOptions{
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return nil, fmt.Errorf("git: open repository %q: %w", repoPath, err)
	}

	state := &LocalGitState{}

	ref, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("git: read HEAD: %w", err)
	}
	state.HeadSHA = ref.Hash().String()

	if ref.Name() == plumbing.HEAD || !ref.Name().IsBranch() {
		state.IsDetached = true
		state.Branch = ref.Hash().String()[:8]
	} else {
		state.Branch = ref.Name().Short()
	}

	wt, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("git: open worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("git: read worktree status: %w", err)
	}
	state.IsClean = status.IsClean()

	for _, s := range status {
		if s.Staging != ' ' && s.Staging != '?' {
			state.StagedCount++
		}
		if s.Worktree != ' ' || s.Staging == '?' {
			state.DirtyCount++
		}
	}

	remotes, err := r.Remotes()
	if err == nil {
		for _, remote := range remotes {
			state.Remotes = append(state.Remotes, remote.Config().Name)
		}
	}

	if remote, err := r.Remote("origin"); err == nil {
		urls := remote.Config().URLs
		if len(urls) > 0 {
			state.DefaultRemote = urls[0]
		}
	}

	ahead, behind := computeDivergence(r, ref)
	state.Ahead = ahead
	state.Behind = behind

	return state, nil
}

func computeDivergence(r *gogit.Repository, headRef *plumbing.Reference) (ahead, behind int) {
	if headRef.Name() == plumbing.HEAD || !headRef.Name().IsBranch() {
		return 0, 0
	}

	branchName := headRef.Name().Short()
	branch, err := r.Branch(branchName)
	if err != nil {
		return 0, 0
	}

	remoteRefName := plumbing.NewRemoteReferenceName(branch.Remote, branch.Merge.String())
	remoteRef, err := r.Reference(remoteRefName, true)
	if err != nil {
		return 0, 0
	}

	localCommit, err := r.CommitObject(headRef.Hash())
	if err != nil {
		return 0, 0
	}
	remoteCommit, err := r.CommitObject(remoteRef.Hash())
	if err != nil {
		return 0, 0
	}

	mergeBase, err := localCommit.MergeBase(remoteCommit)
	if err != nil || len(mergeBase) == 0 {
		return 0, 0
	}

	base := mergeBase[0].Hash

	localIter, err := r.Log(&gogit.LogOptions{From: headRef.Hash()})
	if err != nil {
		return 0, 0
	}
	_ = localIter.ForEach(func(c *object.Commit) error {
		if c.Hash == base {
			return storer.ErrStop
		}
		ahead++
		return nil
	})

	remoteIter, err := r.Log(&gogit.LogOptions{From: remoteRef.Hash()})
	if err != nil {
		return ahead, 0
	}
	_ = remoteIter.ForEach(func(c *object.Commit) error {
		if c.Hash == base {
			return storer.ErrStop
		}
		behind++
		return nil
	})

	return ahead, behind
}
