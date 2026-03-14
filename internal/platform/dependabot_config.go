package platform

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type DependabotConfig struct {
	Version    int                           `json:"version" yaml:"version"`
	Registries map[string]DependabotRegistry `json:"registries,omitempty" yaml:"registries,omitempty"`
	Updates    []DependabotUpdate            `json:"updates,omitempty" yaml:"updates,omitempty"`
}

type DependabotRegistry struct {
	Type     string `json:"type,omitempty" yaml:"type,omitempty"`
	URL      string `json:"url,omitempty" yaml:"url,omitempty"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	Token    string `json:"token,omitempty" yaml:"token,omitempty"`
	Key      string `json:"key,omitempty" yaml:"key,omitempty"`
	Replaces string `json:"replaces_base,omitempty" yaml:"replaces-base,omitempty"`
}

type DependabotUpdate struct {
	Ecosystem       string                     `json:"ecosystem,omitempty"`
	Directories     []string                   `json:"directories,omitempty"`
	Schedule        DependabotSchedule         `json:"schedule,omitempty"`
	GroupedUpdates  map[string]DependabotGroup `json:"grouped_updates,omitempty"`
	VersionUpdates  DependabotVersionPolicy    `json:"version_updates,omitempty"`
	SecurityUpdates DependabotSecurityPolicy   `json:"security_updates,omitempty"`
	OpenPullLimit   int                        `json:"open_pull_limit,omitempty"`
	Labels          []string                   `json:"labels,omitempty"`
	Assignees       []string                   `json:"assignees,omitempty"`
	Registries      []string                   `json:"registries,omitempty"`
	TargetBranch    string                     `json:"target_branch,omitempty"`
	RebaseStrategy  string                     `json:"rebase_strategy,omitempty"`
	CommitMessage   *DependabotCommitMessage   `json:"commit_message,omitempty"`
	Vendor          bool                       `json:"vendor,omitempty"`
}

type DependabotSchedule struct {
	Interval string `json:"interval,omitempty" yaml:"interval,omitempty"`
	Day      string `json:"day,omitempty" yaml:"day,omitempty"`
	Time     string `json:"time,omitempty" yaml:"time,omitempty"`
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`
}

type DependabotGroup struct {
	Patterns        []string `json:"patterns,omitempty" yaml:"patterns,omitempty"`
	DependencyType  string   `json:"dependency_type,omitempty" yaml:"dependency-type,omitempty"`
	UpdateTypes     []string `json:"update_types,omitempty" yaml:"update-types,omitempty"`
	AppliesTo       string   `json:"applies_to,omitempty" yaml:"applies-to,omitempty"`
	ExcludePatterns []string `json:"exclude_patterns,omitempty" yaml:"exclude-patterns,omitempty"`
}

type DependabotVersionPolicy struct {
	Strategy string                 `json:"strategy,omitempty"`
	Allow    []DependabotAllowRule  `json:"allow,omitempty"`
	Ignore   []DependabotIgnoreRule `json:"ignore,omitempty"`
}

type DependabotSecurityPolicy struct {
	GroupedUpdates map[string]DependabotGroup `json:"grouped_updates,omitempty"`
}

type DependabotAllowRule struct {
	DependencyType string `json:"dependency_type,omitempty" yaml:"dependency-type,omitempty"`
	Name           string `json:"name,omitempty" yaml:"dependency-name,omitempty"`
}

type DependabotIgnoreRule struct {
	Name        string   `json:"name,omitempty" yaml:"dependency-name,omitempty"`
	Versions    []string `json:"versions,omitempty" yaml:"versions,omitempty"`
	UpdateTypes []string `json:"update_types,omitempty" yaml:"update-types,omitempty"`
}

type DependabotCommitMessage struct {
	Prefix       string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	PrefixDev    string `json:"prefix_development,omitempty" yaml:"prefix-development,omitempty"`
	IncludeScope string `json:"include_scope,omitempty" yaml:"include,omitempty"`
}

type dependabotYAMLConfig struct {
	Version    int                           `yaml:"version"`
	Registries map[string]DependabotRegistry `yaml:"registries,omitempty"`
	Updates    []dependabotYAMLUpdate        `yaml:"updates,omitempty"`
}

type dependabotYAMLUpdate struct {
	PackageEcosystem      string                     `yaml:"package-ecosystem,omitempty"`
	Directory             string                     `yaml:"directory,omitempty"`
	Schedule              DependabotSchedule         `yaml:"schedule,omitempty"`
	Groups                map[string]DependabotGroup `yaml:"groups,omitempty"`
	OpenPullRequestsLimit int                        `yaml:"open-pull-requests-limit,omitempty"`
	Labels                []string                   `yaml:"labels,omitempty"`
	Assignees             []string                   `yaml:"assignees,omitempty"`
	Registries            []string                   `yaml:"registries,omitempty"`
	TargetBranch          string                     `yaml:"target-branch,omitempty"`
	RebaseStrategy        string                     `yaml:"rebase-strategy,omitempty"`
	CommitMessage         *DependabotCommitMessage   `yaml:"commit-message,omitempty"`
	Vendor                bool                       `yaml:"vendor,omitempty"`
	VersioningStrategy    string                     `yaml:"versioning-strategy,omitempty"`
	Allow                 []DependabotAllowRule      `yaml:"allow,omitempty"`
	Ignore                []DependabotIgnoreRule     `yaml:"ignore,omitempty"`
}

func ParseDependabotConfigYAML(content string) (DependabotConfig, error) {
	var raw dependabotYAMLConfig
	if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return DependabotConfig{}, err
	}
	cfg := DependabotConfig{
		Version:    raw.Version,
		Registries: cloneDependabotRegistries(raw.Registries),
	}
	grouped := map[string]*DependabotUpdate{}
	for _, item := range raw.Updates {
		update := DependabotUpdate{
			Ecosystem:      strings.TrimSpace(item.PackageEcosystem),
			Directories:    compactDependabotStrings([]string{item.Directory}),
			Schedule:       normalizeDependabotSchedule(item.Schedule),
			GroupedUpdates: map[string]DependabotGroup{},
			VersionUpdates: DependabotVersionPolicy{
				Strategy: strings.TrimSpace(item.VersioningStrategy),
				Allow:    cloneDependabotAllow(item.Allow),
				Ignore:   cloneDependabotIgnore(item.Ignore),
			},
			SecurityUpdates: DependabotSecurityPolicy{
				GroupedUpdates: map[string]DependabotGroup{},
			},
			OpenPullLimit:  item.OpenPullRequestsLimit,
			Labels:         compactDependabotStrings(item.Labels),
			Assignees:      compactDependabotStrings(item.Assignees),
			Registries:     compactDependabotStrings(item.Registries),
			TargetBranch:   strings.TrimSpace(item.TargetBranch),
			RebaseStrategy: strings.TrimSpace(item.RebaseStrategy),
			CommitMessage:  cloneDependabotCommitMessage(item.CommitMessage),
			Vendor:         item.Vendor,
		}
		for name, group := range item.Groups {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			normalized := normalizeDependabotGroup(group)
			if strings.EqualFold(normalized.AppliesTo, "security-updates") {
				update.SecurityUpdates.GroupedUpdates[name] = normalized
				continue
			}
			update.GroupedUpdates[name] = normalized
		}
		key := dependabotUpdateKey(update)
		if existing, ok := grouped[key]; ok {
			existing.Directories = compactDependabotStrings(append(existing.Directories, update.Directories...))
			continue
		}
		copyUpdate := cloneDependabotUpdate(update)
		grouped[key] = &copyUpdate
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		cfg.Updates = append(cfg.Updates, cloneDependabotUpdate(*grouped[key]))
	}
	return NormalizeDependabotConfig(cfg), nil
}

func RenderDependabotConfigYAML(cfg DependabotConfig) (string, error) {
	cfg = NormalizeDependabotConfig(cfg)
	if err := ValidateDependabotConfig(cfg); err != nil {
		return "", err
	}
	raw := dependabotYAMLConfig{
		Version:    cfg.Version,
		Registries: cloneDependabotRegistries(cfg.Registries),
	}
	for _, update := range cfg.Updates {
		directories := compactDependabotStrings(update.Directories)
		if len(directories) == 0 {
			directories = []string{"/"}
		}
		for _, directory := range directories {
			entry := dependabotYAMLUpdate{
				PackageEcosystem:      strings.TrimSpace(update.Ecosystem),
				Directory:             strings.TrimSpace(directory),
				Schedule:              normalizeDependabotSchedule(update.Schedule),
				OpenPullRequestsLimit: update.OpenPullLimit,
				Labels:                compactDependabotStrings(update.Labels),
				Assignees:             compactDependabotStrings(update.Assignees),
				Registries:            compactDependabotStrings(update.Registries),
				TargetBranch:          strings.TrimSpace(update.TargetBranch),
				RebaseStrategy:        strings.TrimSpace(update.RebaseStrategy),
				CommitMessage:         cloneDependabotCommitMessage(update.CommitMessage),
				Vendor:                update.Vendor,
				VersioningStrategy:    strings.TrimSpace(update.VersionUpdates.Strategy),
				Allow:                 cloneDependabotAllow(update.VersionUpdates.Allow),
				Ignore:                cloneDependabotIgnore(update.VersionUpdates.Ignore),
			}
			groups := map[string]DependabotGroup{}
			for name, group := range update.GroupedUpdates {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				groups[name] = normalizeDependabotGroup(group)
			}
			for name, group := range update.SecurityUpdates.GroupedUpdates {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				normalized := normalizeDependabotGroup(group)
				if strings.TrimSpace(normalized.AppliesTo) == "" {
					normalized.AppliesTo = "security-updates"
				}
				groups[name] = normalized
			}
			if len(groups) > 0 {
				entry.Groups = groups
			}
			raw.Updates = append(raw.Updates, entry)
		}
	}
	sort.SliceStable(raw.Updates, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(raw.Updates[i].PackageEcosystem + ":" + raw.Updates[i].Directory))
		right := strings.ToLower(strings.TrimSpace(raw.Updates[j].PackageEcosystem + ":" + raw.Updates[j].Directory))
		return left < right
	})
	data, err := yaml.Marshal(raw)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", nil
	}
	return content + "\n", nil
}

func NormalizeDependabotConfig(cfg DependabotConfig) DependabotConfig {
	cfg.Version = maxInt(cfg.Version, 2)
	cfg.Registries = cloneDependabotRegistries(cfg.Registries)
	if len(cfg.Registries) == 0 {
		cfg.Registries = nil
	}
	updates := make([]DependabotUpdate, 0, len(cfg.Updates))
	for _, item := range cfg.Updates {
		update := cloneDependabotUpdate(item)
		update.Ecosystem = strings.TrimSpace(update.Ecosystem)
		update.Directories = compactDependabotStrings(update.Directories)
		update.Schedule = normalizeDependabotSchedule(update.Schedule)
		update.GroupedUpdates = cloneDependabotGroups(update.GroupedUpdates)
		update.SecurityUpdates.GroupedUpdates = cloneDependabotGroups(update.SecurityUpdates.GroupedUpdates)
		update.VersionUpdates.Allow = cloneDependabotAllow(update.VersionUpdates.Allow)
		update.VersionUpdates.Ignore = cloneDependabotIgnore(update.VersionUpdates.Ignore)
		update.Labels = compactDependabotStrings(update.Labels)
		update.Assignees = compactDependabotStrings(update.Assignees)
		update.Registries = compactDependabotStrings(update.Registries)
		update.TargetBranch = strings.TrimSpace(update.TargetBranch)
		update.RebaseStrategy = strings.TrimSpace(update.RebaseStrategy)
		update.VersionUpdates.Strategy = strings.TrimSpace(update.VersionUpdates.Strategy)
		update.CommitMessage = cloneDependabotCommitMessage(update.CommitMessage)
		if update.Ecosystem == "" {
			continue
		}
		if len(update.Directories) == 0 {
			update.Directories = []string{"/"}
		}
		updates = append(updates, update)
	}
	sort.SliceStable(updates, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(updates[i].Ecosystem))
		right := strings.ToLower(strings.TrimSpace(updates[j].Ecosystem))
		if left != right {
			return left < right
		}
		return strings.Join(updates[i].Directories, ",") < strings.Join(updates[j].Directories, ",")
	})
	cfg.Updates = updates
	return cfg
}

func ValidateDependabotConfig(cfg DependabotConfig) error {
	version := cfg.Version
	if version <= 0 {
		version = 2
	}
	if version != 2 {
		return fmt.Errorf("dependabot version must be 2")
	}
	if len(cfg.Updates) == 0 {
		return fmt.Errorf("dependabot config requires at least one update rule")
	}
	for _, update := range cfg.Updates {
		if strings.TrimSpace(update.Ecosystem) == "" {
			return fmt.Errorf("dependabot update ecosystem is required")
		}
		if len(compactDependabotStrings(update.Directories)) == 0 {
			return fmt.Errorf("dependabot update directories are required")
		}
		if strings.TrimSpace(update.Schedule.Interval) == "" {
			return fmt.Errorf("dependabot update schedule interval is required")
		}
	}
	return nil
}

func compactDependabotStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeDependabotSchedule(in DependabotSchedule) DependabotSchedule {
	return DependabotSchedule{
		Interval: strings.TrimSpace(in.Interval),
		Day:      strings.TrimSpace(in.Day),
		Time:     strings.TrimSpace(in.Time),
		Timezone: strings.TrimSpace(in.Timezone),
	}
}

func normalizeDependabotGroup(in DependabotGroup) DependabotGroup {
	return DependabotGroup{
		Patterns:        compactDependabotStrings(in.Patterns),
		DependencyType:  strings.TrimSpace(in.DependencyType),
		UpdateTypes:     compactDependabotStrings(in.UpdateTypes),
		AppliesTo:       strings.TrimSpace(in.AppliesTo),
		ExcludePatterns: compactDependabotStrings(in.ExcludePatterns),
	}
}

func cloneDependabotGroups(in map[string]DependabotGroup) map[string]DependabotGroup {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]DependabotGroup, len(in))
	for key, value := range in {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = normalizeDependabotGroup(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneDependabotRegistries(in map[string]DependabotRegistry) map[string]DependabotRegistry {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]DependabotRegistry, len(in))
	for key, value := range in {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value.Type = strings.TrimSpace(value.Type)
		value.URL = strings.TrimSpace(value.URL)
		value.Username = strings.TrimSpace(value.Username)
		value.Password = strings.TrimSpace(value.Password)
		value.Token = strings.TrimSpace(value.Token)
		value.Key = strings.TrimSpace(value.Key)
		value.Replaces = strings.TrimSpace(value.Replaces)
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneDependabotAllow(in []DependabotAllowRule) []DependabotAllowRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]DependabotAllowRule, 0, len(in))
	for _, item := range in {
		item.DependencyType = strings.TrimSpace(item.DependencyType)
		item.Name = strings.TrimSpace(item.Name)
		if item.DependencyType == "" && item.Name == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func cloneDependabotIgnore(in []DependabotIgnoreRule) []DependabotIgnoreRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]DependabotIgnoreRule, 0, len(in))
	for _, item := range in {
		item.Name = strings.TrimSpace(item.Name)
		item.Versions = compactDependabotStrings(item.Versions)
		item.UpdateTypes = compactDependabotStrings(item.UpdateTypes)
		if item.Name == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func cloneDependabotCommitMessage(in *DependabotCommitMessage) *DependabotCommitMessage {
	if in == nil {
		return nil
	}
	return &DependabotCommitMessage{
		Prefix:       strings.TrimSpace(in.Prefix),
		PrefixDev:    strings.TrimSpace(in.PrefixDev),
		IncludeScope: strings.TrimSpace(in.IncludeScope),
	}
}

func cloneDependabotUpdate(in DependabotUpdate) DependabotUpdate {
	return DependabotUpdate{
		Ecosystem:      strings.TrimSpace(in.Ecosystem),
		Directories:    compactDependabotStrings(in.Directories),
		Schedule:       normalizeDependabotSchedule(in.Schedule),
		GroupedUpdates: cloneDependabotGroups(in.GroupedUpdates),
		VersionUpdates: DependabotVersionPolicy{
			Strategy: strings.TrimSpace(in.VersionUpdates.Strategy),
			Allow:    cloneDependabotAllow(in.VersionUpdates.Allow),
			Ignore:   cloneDependabotIgnore(in.VersionUpdates.Ignore),
		},
		SecurityUpdates: DependabotSecurityPolicy{
			GroupedUpdates: cloneDependabotGroups(in.SecurityUpdates.GroupedUpdates),
		},
		OpenPullLimit:  in.OpenPullLimit,
		Labels:         compactDependabotStrings(in.Labels),
		Assignees:      compactDependabotStrings(in.Assignees),
		Registries:     compactDependabotStrings(in.Registries),
		TargetBranch:   strings.TrimSpace(in.TargetBranch),
		RebaseStrategy: strings.TrimSpace(in.RebaseStrategy),
		CommitMessage:  cloneDependabotCommitMessage(in.CommitMessage),
		Vendor:         in.Vendor,
	}
}

func dependabotUpdateKey(update DependabotUpdate) string {
	update = cloneDependabotUpdate(update)
	update.Directories = nil
	return strings.TrimSpace(mustJSON(update))
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
