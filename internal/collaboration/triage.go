package collaboration

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// TriagePriority represents triage priority levels.
type TriagePriority string

const (
	TriageCritical      TriagePriority = "critical"
	TriageHigh          TriagePriority = "high"
	TriageMedium        TriagePriority = "medium"
	TriageLow           TriagePriority = "low"
	TriageInformational TriagePriority = "informational"
)

// TriageResult holds the result of triaging an object.
type TriageResult struct {
	ObjectRef       string         `json:"object_ref" yaml:"object_ref"`
	Priority        TriagePriority `json:"priority" yaml:"priority"`
	Reason          string         `json:"reason" yaml:"reason"`
	SuggestedAction string         `json:"suggested_action" yaml:"suggested_action"`
	Tags            []string       `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// ActivitySummary summarizes activity over a period.
type ActivitySummary struct {
	RepoOwner    string         `json:"repo_owner" yaml:"repo_owner"`
	RepoName     string         `json:"repo_name" yaml:"repo_name"`
	Period       string         `json:"period" yaml:"period"`
	TotalObjects int            `json:"total_objects" yaml:"total_objects"`
	ByType       map[string]int `json:"by_type" yaml:"by_type"`
	ByPriority   map[string]int `json:"by_priority" yaml:"by_priority"`
	TopItems     []TriageResult `json:"top_items" yaml:"top_items"`
	GeneratedAt  time.Time      `json:"generated_at" yaml:"generated_at"`
}

// TriageEngine triages and summarizes collaboration objects.
type TriageEngine interface {
	Triage(ctx context.Context, object *CollaborationObject) (*TriageResult, error)
	Summarize(ctx context.Context, objects []*CollaborationObject, period string) (*ActivitySummary, error)
}

// RuleBasedTriageEngine implements TriageEngine with simple label-based rules.
type RuleBasedTriageEngine struct{}

// NewRuleBasedTriageEngine creates a new RuleBasedTriageEngine.
func NewRuleBasedTriageEngine() *RuleBasedTriageEngine {
	return &RuleBasedTriageEngine{}
}

// Triage assigns priority to an object based on labels and state.
func (e *RuleBasedTriageEngine) Triage(_ context.Context, obj *CollaborationObject) (*TriageResult, error) {
	if obj == nil {
		return nil, fmt.Errorf("object cannot be nil")
	}

	ref := ObjectRef(obj)
	labels := make([]string, len(obj.Labels))
	copy(labels, obj.Labels)
	for i := range labels {
		labels[i] = strings.ToLower(labels[i])
	}

	priority := TriageMedium
	reason := "default"
	suggestedAction := "review"
	tags := []string{string(obj.ObjectType), obj.State}

	// security label = critical
	if hasLabel(labels, "security") {
		priority = TriageCritical
		reason = "security label"
		suggestedAction = "address immediately"
		tags = append(tags, "security")
	} else if hasLabel(labels, "bug") {
		priority = TriageHigh
		reason = "bug label"
		suggestedAction = "investigate and fix"
		tags = append(tags, "bug")
	} else if hasLabel(labels, "stale") || hasLabel(labels, "wontfix") {
		priority = TriageLow
		reason = "stale or wontfix"
		suggestedAction = "consider closing"
		tags = append(tags, "stale")
	} else if hasLabel(labels, "documentation") || hasLabel(labels, "question") {
		priority = TriageInformational
		reason = "documentation or question"
		suggestedAction = "respond when convenient"
		tags = append(tags, "docs")
	} else if obj.State == "closed" {
		priority = TriageLow
		reason = "closed"
		suggestedAction = "no action"
	}

	return &TriageResult{
		ObjectRef:       ref,
		Priority:        priority,
		Reason:          reason,
		SuggestedAction: suggestedAction,
		Tags:            tags,
	}, nil
}

// Summarize builds an ActivitySummary from triaged objects.
func (e *RuleBasedTriageEngine) Summarize(ctx context.Context, objects []*CollaborationObject, period string) (*ActivitySummary, error) {
	byType := make(map[string]int)
	byPriority := make(map[string]int)
	var results []TriageResult

	for _, obj := range objects {
		res, err := e.Triage(ctx, obj)
		if err != nil {
			continue
		}
		byType[string(obj.ObjectType)]++
		byPriority[string(res.Priority)]++
		results = append(results, *res)
	}

	// Sort by priority (critical first) and take top 10
	sort.Slice(results, func(i, j int) bool {
		return priorityOrder(results[i].Priority) < priorityOrder(results[j].Priority)
	})
	topN := 10
	if len(results) < topN {
		topN = len(results)
	}
	topItems := make([]TriageResult, topN)
	copy(topItems, results[:topN])

	owner, repo := "", ""
	if len(objects) > 0 {
		owner = objects[0].RepoOwner
		repo = objects[0].RepoName
	}

	return &ActivitySummary{
		RepoOwner:    owner,
		RepoName:     repo,
		Period:       period,
		TotalObjects: len(objects),
		ByType:       byType,
		ByPriority:   byPriority,
		TopItems:     topItems,
		GeneratedAt:  time.Now().UTC(),
	}, nil
}

func hasLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}

func priorityOrder(p TriagePriority) int {
	switch p {
	case TriageCritical:
		return 0
	case TriageHigh:
		return 1
	case TriageMedium:
		return 2
	case TriageLow:
		return 3
	case TriageInformational:
		return 4
	default:
		return 5
	}
}

// ObjectRef returns the canonical ref for a collaboration object.
func ObjectRef(obj *CollaborationObject) string {
	if obj == nil {
		return ""
	}
	if obj.ObjectType == ObjectTypePullRequest {
		return fmt.Sprintf("%s/%s#pr/%d", obj.RepoOwner, obj.RepoName, obj.Number)
	}
	return fmt.Sprintf("%s/%s#%d", obj.RepoOwner, obj.RepoName, obj.Number)
}
