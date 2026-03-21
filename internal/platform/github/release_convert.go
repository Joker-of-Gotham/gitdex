package github

import (
	"time"

	gh "github.com/google/go-github/v84/github"
)

func releaseFromGH(r *gh.RepositoryRelease) *Release {
	if r == nil {
		return nil
	}
	out := &Release{
		ID:         r.GetID(),
		TagName:    r.GetTagName(),
		Name:       r.GetName(),
		Body:       r.GetBody(),
		Draft:      r.GetDraft(),
		Prerelease: r.GetPrerelease(),
		HTMLURL:    r.GetHTMLURL(),
	}
	if r.CreatedAt != nil {
		out.CreatedAt = r.CreatedAt.Time
	} else {
		out.CreatedAt = time.Time{}
	}
	if r.PublishedAt != nil {
		out.PublishedAt = r.PublishedAt.Time
	} else {
		out.PublishedAt = time.Time{}
	}
	return out
}
