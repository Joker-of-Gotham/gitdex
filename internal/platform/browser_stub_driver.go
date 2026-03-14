package platform

import (
	"encoding/json"
	"strings"
)

type BrowserStubDriver interface {
	Name() string
	StubState(capabilityID, flow, resourceID string) json.RawMessage
}

type staticBrowserStubDriver struct {
	name string
}

var browserStubDriverAliases = map[string]string{
	"":                  "default",
	"default":           "default",
	"playwright":        "playwright",
	"selenium":          "selenium",
	"stub-driver":       "default",
	"github-browser":    "github-browser",
	"gitlab-browser":    "gitlab-browser",
	"bitbucket-browser": "bitbucket-browser",
}

func ResolveBrowserStubDriver(name string) BrowserStubDriver {
	name = strings.ToLower(strings.TrimSpace(name))
	if canonical, ok := browserStubDriverAliases[name]; ok {
		name = canonical
	}
	if name == "" {
		name = "default"
	}
	return staticBrowserStubDriver{name: name}
}

func (d staticBrowserStubDriver) Name() string {
	return strings.TrimSpace(firstNonEmptyAdmin(d.name, "default"))
}

func (d staticBrowserStubDriver) StubState(capabilityID, flow, resourceID string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"adapter":                    AdapterBrowser,
		"browser_driver":             d.Name(),
		"capability_id":              strings.TrimSpace(capabilityID),
		"flow":                       strings.TrimSpace(flow),
		"resource_id":                strings.TrimSpace(resourceID),
		"manual_completion_required": true,
	})
	return raw
}
