package collaboration

import (
	"context"
	"fmt"
	"time"

	gh "github.com/google/go-github/v84/github"
	ghp "github.com/your-org/gitdex/internal/platform/github"
)

// MutationType represents the type of collaboration mutation.
type MutationType string

const (
	MutationCreate  MutationType = "create"
	MutationUpdate  MutationType = "update"
	MutationComment MutationType = "comment"
	MutationClose   MutationType = "close"
	MutationReopen  MutationType = "reopen"
	MutationLabel   MutationType = "label"
	MutationAssign  MutationType = "assign"
	MutationMerge   MutationType = "merge"
)

// MutationRequest represents a request to mutate a collaboration object.
type MutationRequest struct {
	MutationType MutationType `json:"mutation_type" yaml:"mutation_type"`
	ObjectType   ObjectType   `json:"object_type" yaml:"object_type"`
	RepoOwner    string       `json:"repo_owner" yaml:"repo_owner"`
	RepoName     string       `json:"repo_name" yaml:"repo_name"`
	Number       *int         `json:"number,omitempty" yaml:"number,omitempty"`
	Title        string       `json:"title,omitempty" yaml:"title,omitempty"`
	Body         string       `json:"body,omitempty" yaml:"body,omitempty"`
	Labels       []string     `json:"labels,omitempty" yaml:"labels,omitempty"`
	Assignees    []string     `json:"assignees,omitempty" yaml:"assignees,omitempty"`
}

// MutationResult represents the result of a mutation.
type MutationResult struct {
	Request MutationRequest      `json:"request" yaml:"request"`
	Success bool                 `json:"success" yaml:"success"`
	Object  *CollaborationObject `json:"object,omitempty" yaml:"object,omitempty"`
	Message string               `json:"message" yaml:"message"`
}

// MutationEngine executes mutations on collaboration objects.
type MutationEngine interface {
	Execute(ctx context.Context, request *MutationRequest) (*MutationResult, error)
}

// GitHubMutationEngine executes mutations via the real GitHub API.
type GitHubMutationEngine struct {
	client *ghp.Client
}

// NewGitHubMutationEngine creates a new GitHubMutationEngine.
func NewGitHubMutationEngine(client *ghp.Client) *GitHubMutationEngine {
	return &GitHubMutationEngine{client: client}
}

// Execute executes a mutation request via the GitHub API.
func (e *GitHubMutationEngine) Execute(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if e.client == nil {
		return nil, fmt.Errorf("GitHub client is required; run 'gitdex setup' to configure")
	}
	if req == nil {
		return &MutationResult{Success: false, Message: "request cannot be nil"}, nil
	}
	switch req.MutationType {
	case MutationCreate:
		return e.executeCreate(ctx, req)
	case MutationComment:
		return e.executeComment(ctx, req)
	case MutationClose:
		return e.executeClose(ctx, req)
	case MutationReopen:
		return e.executeReopen(ctx, req)
	case MutationUpdate:
		return e.executeUpdate(ctx, req)
	case MutationLabel:
		return e.executeLabel(ctx, req)
	case MutationAssign:
		return e.executeAssign(ctx, req)
	case MutationMerge:
		return e.executeMerge(ctx, req)
	default:
		return &MutationResult{
			Request: *req,
			Success: false,
			Message: fmt.Sprintf("unsupported mutation type: %s", req.MutationType),
		}, nil
	}
}

func issueToObject(issue *gh.Issue, owner, repo string) *CollaborationObject {
	if issue == nil {
		return nil
	}
	labels := make([]string, 0, len(issue.Labels))
	for _, l := range issue.Labels {
		labels = append(labels, l.GetName())
	}
	assignees := make([]string, 0, len(issue.Assignees))
	for _, a := range issue.Assignees {
		assignees = append(assignees, a.GetLogin())
	}
	var createdAt, updatedAt time.Time
	if issue.CreatedAt != nil {
		createdAt = issue.CreatedAt.Time
	}
	if issue.UpdatedAt != nil {
		updatedAt = issue.UpdatedAt.Time
	}
	return &CollaborationObject{
		ObjectID:      issue.GetNodeID(),
		ObjectType:    ObjectTypeIssue,
		RepoOwner:     owner,
		RepoName:      repo,
		Number:        issue.GetNumber(),
		Title:         issue.GetTitle(),
		State:         issue.GetState(),
		Author:        issue.GetUser().GetLogin(),
		Assignees:     assignees,
		Labels:        labels,
		Body:          issue.GetBody(),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		CommentsCount: issue.GetComments(),
		URL:           issue.GetHTMLURL(),
	}
}

func (e *GitHubMutationEngine) executeCreate(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if req.ObjectType == ObjectTypeDiscussion {
		discussion, err := e.client.CreateDiscussion(ctx, req.RepoOwner, req.RepoName, req.Title, req.Body)
		if err != nil {
			return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
		}
		obj := &CollaborationObject{
			ObjectID:      discussion.ID,
			ObjectType:    ObjectTypeDiscussion,
			RepoOwner:     req.RepoOwner,
			RepoName:      req.RepoName,
			Number:        discussion.Number,
			Title:         discussion.Title,
			State:         discussion.State,
			Author:        discussion.Author,
			Body:          discussion.Body,
			CommentsCount: discussion.CommentsCount,
			URL:           discussion.URL,
		}
		return &MutationResult{Request: *req, Success: true, Object: obj, Message: "created"}, nil
	}
	issue, err := e.client.CreateIssue(ctx, req.RepoOwner, req.RepoName, req.Title, req.Body, req.Labels, req.Assignees)
	if err != nil {
		return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
	}
	obj := issueToObject(issue, req.RepoOwner, req.RepoName)
	return &MutationResult{Request: *req, Success: true, Object: obj, Message: "created"}, nil
}

func (e *GitHubMutationEngine) executeComment(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if req.Number == nil {
		return &MutationResult{Request: *req, Success: false, Message: "number required for comment"}, nil
	}
	if req.ObjectType == ObjectTypeDiscussion {
		if err := e.client.AddDiscussionComment(ctx, req.RepoOwner, req.RepoName, *req.Number, req.Body); err != nil {
			return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
		}
		return &MutationResult{Request: *req, Success: true, Message: "comment added"}, nil
	}
	_, err := e.client.CreateComment(ctx, req.RepoOwner, req.RepoName, *req.Number, req.Body)
	if err != nil {
		return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
	}
	return &MutationResult{Request: *req, Success: true, Message: "comment added"}, nil
}

func (e *GitHubMutationEngine) executeClose(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if req.Number == nil {
		return &MutationResult{Request: *req, Success: false, Message: "number required for close"}, nil
	}
	if req.ObjectType == ObjectTypeDiscussion {
		if err := e.client.CloseDiscussion(ctx, req.RepoOwner, req.RepoName, *req.Number); err != nil {
			return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
		}
		return &MutationResult{Request: *req, Success: true, Message: "closed"}, nil
	}
	if err := e.client.CloseIssue(ctx, req.RepoOwner, req.RepoName, *req.Number); err != nil {
		return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
	}
	return &MutationResult{Request: *req, Success: true, Message: "closed"}, nil
}

func (e *GitHubMutationEngine) executeReopen(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if req.Number == nil {
		return &MutationResult{Request: *req, Success: false, Message: "number required for reopen"}, nil
	}
	if req.ObjectType == ObjectTypeDiscussion {
		if err := e.client.ReopenDiscussion(ctx, req.RepoOwner, req.RepoName, *req.Number); err != nil {
			return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
		}
		return &MutationResult{Request: *req, Success: true, Message: "reopened"}, nil
	}
	if err := e.client.ReopenIssue(ctx, req.RepoOwner, req.RepoName, *req.Number); err != nil {
		return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
	}
	return &MutationResult{Request: *req, Success: true, Message: "reopened"}, nil
}

func (e *GitHubMutationEngine) executeUpdate(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if req.Number == nil {
		return &MutationResult{Request: *req, Success: false, Message: "number required for update"}, nil
	}
	ir := &gh.IssueRequest{}
	if req.Title != "" {
		ir.Title = gh.String(req.Title)
	}
	if req.Body != "" {
		ir.Body = gh.String(req.Body)
	}
	if len(req.Labels) > 0 {
		ir.Labels = &req.Labels
	}
	if len(req.Assignees) > 0 {
		ir.Assignees = &req.Assignees
	}
	if ir.Title == nil && ir.Body == nil && ir.Labels == nil && ir.Assignees == nil {
		return &MutationResult{Request: *req, Success: true, Message: "nothing to update"}, nil
	}
	issue, err := e.client.UpdateIssue(ctx, req.RepoOwner, req.RepoName, *req.Number, ir)
	if err != nil {
		return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
	}
	obj := issueToObject(issue, req.RepoOwner, req.RepoName)
	return &MutationResult{Request: *req, Success: true, Object: obj, Message: "updated"}, nil
}

func (e *GitHubMutationEngine) executeLabel(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if req.Number == nil {
		return &MutationResult{Request: *req, Success: false, Message: "number required for label"}, nil
	}
	if len(req.Labels) == 0 {
		return &MutationResult{Request: *req, Success: false, Message: "labels required"}, nil
	}
	if err := e.client.AddLabels(ctx, req.RepoOwner, req.RepoName, *req.Number, req.Labels); err != nil {
		return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
	}
	return &MutationResult{Request: *req, Success: true, Message: "labels added"}, nil
}

func (e *GitHubMutationEngine) executeAssign(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if req.Number == nil {
		return &MutationResult{Request: *req, Success: false, Message: "number required for assign"}, nil
	}
	if err := e.client.SetAssignees(ctx, req.RepoOwner, req.RepoName, *req.Number, req.Assignees); err != nil {
		return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
	}
	return &MutationResult{Request: *req, Success: true, Message: "assignees set"}, nil
}

func (e *GitHubMutationEngine) executeMerge(ctx context.Context, req *MutationRequest) (*MutationResult, error) {
	if req.Number == nil {
		return &MutationResult{Request: *req, Success: false, Message: "number required for merge"}, nil
	}
	method := "merge"
	if req.Body != "" {
		method = req.Body
	}
	_, err := e.client.MergePullRequest(ctx, req.RepoOwner, req.RepoName, *req.Number, "", method)
	if err != nil {
		return &MutationResult{Request: *req, Success: false, Message: err.Error()}, nil
	}
	return &MutationResult{Request: *req, Success: true, Message: "merged"}, nil
}
