package platform

import "context"

type Platform int

const (
	PlatformGitHub Platform = iota
	PlatformGitLab
	PlatformBitbucket
	PlatformUnknown
)

type PRRequest struct {
	Title      string
	Body       string
	BaseBranch string
	HeadBranch string
	Draft      bool
}

type PRResponse struct {
	URL    string
	Number int
}

type IssueFilter struct {
	State  string
	Labels []string
}

type Issue struct {
	Number int
	Title  string
	URL    string
}

type ContributingSpec struct {
	CommitConvention string
	BranchNaming     string
	PRTemplate       string
	DCORequired      bool
}

type PlatformAdapter interface {
	DetectPlatform(ctx context.Context, remoteURL string) (Platform, error)
	CreatePR(ctx context.Context, req PRRequest) (*PRResponse, error)
	ListIssues(ctx context.Context, filter IssueFilter) ([]Issue, error)
	GetContributing(ctx context.Context) (*ContributingSpec, error)
}
