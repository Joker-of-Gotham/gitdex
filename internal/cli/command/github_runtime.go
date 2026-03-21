package command

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
	"github.com/your-org/gitdex/internal/platform/identity"
)

var githubClientFactoryOverride func(app bootstrap.App) (*ghclient.Client, error)

func SetGitHubClientFactoryForTest(factory func(app bootstrap.App) (*ghclient.Client, error)) func() {
	prev := githubClientFactoryOverride
	githubClientFactoryOverride = factory
	return func() { githubClientFactoryOverride = prev }
}

func newGitHubClientFromApp(app bootstrap.App) (*ghclient.Client, error) {
	if githubClientFactoryOverride != nil {
		return githubClientFactoryOverride(app)
	}
	tr, err := identity.ResolveTransport(app.Config.Identity, http.DefaultTransport)
	if err != nil {
		if errors.Is(err, identity.ErrNoIdentity) {
			return nil, nil
		}
		return nil, err
	}

	httpClient := &http.Client{Transport: tr.Transport}
	if tr.Host != "" && tr.Host != "github.com" {
		return ghclient.NewClientWithBaseURL(httpClient, fmt.Sprintf("https://%s/api/v3", tr.Host))
	}
	return ghclient.NewClient(httpClient), nil
}
