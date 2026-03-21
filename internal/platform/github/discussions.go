package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Discussion struct {
	ID            string
	Number        int
	Title         string
	Body          string
	State         string
	URL           string
	Author        string
	Category      string
	CommentsCount int
	CreatedAt     string
	UpdatedAt     string
}

func (c *Client) ListDiscussions(ctx context.Context, owner, repo string) ([]Discussion, error) {
	var payload struct {
		Repository struct {
			Discussions struct {
				Nodes []discussionNode `json:"nodes"`
			} `json:"discussions"`
		} `json:"repository"`
	}
	const query = `
query($owner:String!, $repo:String!, $first:Int!) {
  repository(owner:$owner, name:$repo) {
    discussions(first:$first, orderBy:{field:UPDATED_AT, direction:DESC}) {
      nodes {
        id
        number
        title
        body
        state
        url
        createdAt
        updatedAt
        category { name }
        author { login }
        comments(first:1) { totalCount }
      }
    }
  }
}`
	if err := c.doGraphQL(ctx, query, map[string]any{
		"owner": owner,
		"repo":  repo,
		"first": 50,
	}, &payload); err != nil {
		return nil, err
	}
	result := make([]Discussion, 0, len(payload.Repository.Discussions.Nodes))
	for _, node := range payload.Repository.Discussions.Nodes {
		result = append(result, node.toDiscussion())
	}
	return result, nil
}

func (c *Client) GetDiscussion(ctx context.Context, owner, repo string, number int) (*Discussion, error) {
	var payload struct {
		Repository struct {
			Discussion *discussionNode `json:"discussion"`
		} `json:"repository"`
	}
	const query = `
query($owner:String!, $repo:String!, $number:Int!) {
  repository(owner:$owner, name:$repo) {
    discussion(number:$number) {
      id
      number
      title
      body
      state
      url
      createdAt
      updatedAt
      category { name }
      author { login }
      comments(first:1) { totalCount }
    }
  }
}`
	if err := c.doGraphQL(ctx, query, map[string]any{
		"owner":  owner,
		"repo":   repo,
		"number": number,
	}, &payload); err != nil {
		return nil, err
	}
	if payload.Repository.Discussion == nil {
		return nil, fmt.Errorf("github: discussion #%d not found", number)
	}
	discussion := payload.Repository.Discussion.toDiscussion()
	return &discussion, nil
}

func (c *Client) CreateDiscussion(ctx context.Context, owner, repo, title, body string) (*Discussion, error) {
	repositoryID, categoryID, err := c.resolveDiscussionCategory(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	var payload struct {
		CreateDiscussion struct {
			Discussion discussionNode `json:"discussion"`
		} `json:"createDiscussion"`
	}
	const mutation = `
mutation($repositoryId:ID!, $categoryId:ID!, $title:String!, $body:String!) {
  createDiscussion(input:{repositoryId:$repositoryId, categoryId:$categoryId, title:$title, body:$body}) {
    discussion {
      id
      number
      title
      body
      state
      url
      createdAt
      updatedAt
      category { name }
      author { login }
      comments(first:1) { totalCount }
    }
  }
}`
	if err := c.doGraphQL(ctx, mutation, map[string]any{
		"repositoryId": repositoryID,
		"categoryId":   categoryID,
		"title":        title,
		"body":         body,
	}, &payload); err != nil {
		return nil, err
	}
	discussion := payload.CreateDiscussion.Discussion.toDiscussion()
	return &discussion, nil
}

func (c *Client) AddDiscussionComment(ctx context.Context, owner, repo string, number int, body string) error {
	discussion, err := c.GetDiscussion(ctx, owner, repo, number)
	if err != nil {
		return err
	}
	var payload struct {
		AddDiscussionComment struct {
			Comment struct {
				ID string `json:"id"`
			} `json:"comment"`
		} `json:"addDiscussionComment"`
	}
	const mutation = `
mutation($discussionId:ID!, $body:String!) {
  addDiscussionComment(input:{discussionId:$discussionId, body:$body}) {
    comment { id }
  }
}`
	if err := c.doGraphQL(ctx, mutation, map[string]any{
		"discussionId": discussion.ID,
		"body":         body,
	}, &payload); err != nil {
		return err
	}
	if payload.AddDiscussionComment.Comment.ID == "" {
		return fmt.Errorf("github: discussion comment was not created")
	}
	return nil
}

func (c *Client) CloseDiscussion(ctx context.Context, owner, repo string, number int) error {
	discussion, err := c.GetDiscussion(ctx, owner, repo, number)
	if err != nil {
		return err
	}
	var payload struct {
		CloseDiscussion struct {
			Discussion struct {
				ID string `json:"id"`
			} `json:"discussion"`
		} `json:"closeDiscussion"`
	}
	const mutation = `
mutation($discussionId:ID!) {
  closeDiscussion(input:{discussionId:$discussionId}) {
    discussion { id }
  }
}`
	if err := c.doGraphQL(ctx, mutation, map[string]any{"discussionId": discussion.ID}, &payload); err != nil {
		return err
	}
	return nil
}

func (c *Client) ReopenDiscussion(ctx context.Context, owner, repo string, number int) error {
	discussion, err := c.GetDiscussion(ctx, owner, repo, number)
	if err != nil {
		return err
	}
	var payload struct {
		ReopenDiscussion struct {
			Discussion struct {
				ID string `json:"id"`
			} `json:"discussion"`
		} `json:"reopenDiscussion"`
	}
	const mutation = `
mutation($discussionId:ID!) {
  reopenDiscussion(input:{discussionId:$discussionId}) {
    discussion { id }
  }
}`
	if err := c.doGraphQL(ctx, mutation, map[string]any{"discussionId": discussion.ID}, &payload); err != nil {
		return err
	}
	return nil
}

func (c *Client) resolveDiscussionCategory(ctx context.Context, owner, repo string) (string, string, error) {
	var payload struct {
		Repository struct {
			ID                   string `json:"id"`
			DiscussionCategories struct {
				Nodes []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"nodes"`
			} `json:"discussionCategories"`
		} `json:"repository"`
	}
	const query = `
query($owner:String!, $repo:String!) {
  repository(owner:$owner, name:$repo) {
    id
    discussionCategories(first:10) {
      nodes {
        id
        name
      }
    }
  }
}`
	if err := c.doGraphQL(ctx, query, map[string]any{
		"owner": owner,
		"repo":  repo,
	}, &payload); err != nil {
		return "", "", err
	}
	if payload.Repository.ID == "" {
		return "", "", fmt.Errorf("github: repository %s/%s not found", owner, repo)
	}
	for _, category := range payload.Repository.DiscussionCategories.Nodes {
		if category.ID != "" {
			return payload.Repository.ID, category.ID, nil
		}
	}
	return "", "", fmt.Errorf("github: repository %s/%s has no discussion categories enabled", owner, repo)
}

func (c *Client) doGraphQL(ctx context.Context, query string, variables map[string]any, dest any) error {
	if c.httpClient == nil {
		return fmt.Errorf("github: graphql client unavailable")
	}
	requestBody, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return fmt.Errorf("github: marshal graphql request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.graphQLEndpoint, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("github: build graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("github: execute graphql request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("github: read graphql response: %w", err)
	}
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("github: decode graphql response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("github: graphql error: %s", envelope.Errors[0].Message)
	}
	if len(envelope.Data) == 0 {
		return fmt.Errorf("github: graphql response missing data")
	}
	if err := json.Unmarshal(envelope.Data, dest); err != nil {
		return fmt.Errorf("github: decode graphql payload: %w", err)
	}
	return nil
}

type discussionNode struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Category  struct {
		Name string `json:"name"`
	} `json:"category"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	Comments struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
}

func (n discussionNode) toDiscussion() Discussion {
	return Discussion{
		ID:            n.ID,
		Number:        n.Number,
		Title:         n.Title,
		Body:          n.Body,
		State:         strings.ToLower(n.State),
		URL:           n.URL,
		Author:        n.Author.Login,
		Category:      n.Category.Name,
		CommentsCount: n.Comments.TotalCount,
		CreatedAt:     n.CreatedAt,
		UpdatedAt:     n.UpdatedAt,
	}
}
