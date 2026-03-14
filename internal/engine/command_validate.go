package engine

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

func normalizeSuggestionAgainstState(s git.Suggestion, state *status.GitState) (git.Suggestion, bool) {
	if state == nil {
		return s, true
	}

	if s.Interaction == git.FileWrite && s.FileOp != nil {
		op := strings.ToLower(strings.TrimSpace(s.FileOp.Operation))
		if op == "delete" && !repoPathExists(s.FileOp.Path) {
			return s, false
		}
		return s, true
	}

	if len(s.Command) < 2 || !strings.EqualFold(strings.TrimSpace(s.Command[0]), "git") {
		return s, true
	}

	argv := append([]string(nil), s.Command...)
	sub := strings.ToLower(strings.TrimSpace(argv[1]))

	switch sub {
	case "commit":
		if len(state.StagingArea) == 0 {
			return s, false
		}

	case "rm":
		for _, arg := range positionalArgs(argv[2:]) {
			if !repoPathExists(arg) {
				return s, false
			}
		}

	case "add":
		if len(state.WorkingTree) == 0 {
			return s, false
		}
		for _, arg := range positionalArgs(argv[2:]) {
			if arg == "." {
				continue
			}
			if !pathMentionedInWorkingTree(state, arg) && !repoPathExists(arg) {
				return s, false
			}
		}

	case "push", "pull", "fetch":
		if !validateNetworkCommand(sub, argv, state) {
			return s, false
		}

	case "checkout":
		next, ok := normalizeCheckoutSuggestion(s, state)
		return next, ok

	case "switch":
		if !validateSwitchSuggestion(argv, state) {
			return s, false
		}

	case "branch":
		if !validateBranchSuggestion(argv, state) {
			return s, false
		}

	case "remote":
		if !validateRemoteSuggestion(argv, state) {
			return s, false
		}

	case "check-ignore":
		args := positionalArgs(argv[2:])
		if len(args) == 0 {
			return s, false
		}
		for _, arg := range args {
			if !repoPathExists(arg) {
				return s, false
			}
		}

	case "tag":
		if !validateTagSuggestion(argv, state) {
			return s, false
		}

	case "restore":
		if !validateRestoreSuggestion(argv, state) {
			return s, false
		}
	}

	s.Command = argv
	return s, true
}

func validateNetworkCommand(sub string, argv []string, state *status.GitState) bool {
	positional := positionalArgs(argv[2:])
	if len(positional) == 0 {
		if sub == "push" || sub == "pull" {
			return state.LocalBranch.Upstream != "" || len(state.RemoteInfos) > 0
		}
		return len(state.RemoteInfos) > 0
	}

	first := positional[0]
	if isLikelyURL(first) {
		return true
	}
	if remoteExists(state, first) {
		return true
	}
	if sub == "push" || sub == "pull" {
		if first == currentBranch(state) {
			return state.LocalBranch.Upstream != ""
		}
	}
	return false
}

func normalizeCheckoutSuggestion(s git.Suggestion, state *status.GitState) (git.Suggestion, bool) {
	argv := append([]string(nil), s.Command...)
	if len(argv) < 3 {
		return s, true
	}

	if containsToken(argv[2:], "--") {
		files := argsAfterDoubleDash(argv[2:])
		if len(files) == 0 {
			return s, false
		}
		for _, file := range files {
			if !repoPathExists(file) {
				return s, false
			}
		}
		s.Command = argv
		return s, true
	}

	if createTarget, ok := flagValue(argv[2:], "-b", "-B"); ok {
		if branchExists(state, createTarget) {
			return s, false
		}
		s.Command = argv
		return s, true
	}

	positional := positionalArgs(argv[2:])
	if len(positional) == 0 {
		return s, true
	}
	if len(positional) == 1 {
		target := positional[0]
		if localBranchExists(state, target) {
			if strings.EqualFold(currentBranch(state), target) {
				return s, false
			}
			s.Command = []string{"git", "switch", target}
			return s, true
		}
		if remoteBranchExists(state, target) {
			s.Command = []string{"git", "switch", "--track", target}
			return s, true
		}
		if repoPathExists(target) {
			return s, true
		}
		return s, false
	}

	for _, target := range positional {
		if !repoPathExists(target) {
			return s, false
		}
	}
	s.Command = argv
	return s, true
}

func validateSwitchSuggestion(argv []string, state *status.GitState) bool {
	if len(argv) < 3 {
		return true
	}

	if createTarget, ok := flagValue(argv[2:], "-c", "-C"); ok {
		return !branchExists(state, createTarget)
	}

	positional := positionalArgs(argv[2:])
	if len(positional) == 0 {
		return true
	}
	target := positional[0]
	if strings.EqualFold(target, currentBranch(state)) {
		return false
	}
	return localBranchExists(state, target) || remoteBranchExists(state, target) || looksLikeCommitish(target)
}

func validateBranchSuggestion(argv []string, state *status.GitState) bool {
	if len(argv) < 3 {
		return true
	}
	for _, arg := range argv[2:] {
		switch strings.TrimSpace(arg) {
		case "--list", "-a", "-r", "-vv", "--contains", "--merged", "--no-merged", "--show-current":
			return true
		}
	}
	if target, ok := flagValue(argv[2:], "-d", "-D"); ok {
		return localBranchExists(state, target) && !strings.EqualFold(currentBranch(state), target)
	}
	if strings.HasPrefix(strings.TrimSpace(argv[2]), "-") {
		return true
	}

	positional := positionalArgs(argv[2:])
	if len(positional) == 0 {
		return true
	}

	target := positional[0]
	if strings.EqualFold(target, currentBranch(state)) {
		return false
	}
	return !localBranchExists(state, target)
}

func validateRemoteSuggestion(argv []string, state *status.GitState) bool {
	args := positionalArgs(argv[2:])
	if len(args) == 0 {
		return true
	}

	switch strings.ToLower(args[0]) {
	case "add":
		if len(args) < 3 {
			return false
		}
		return !remoteExists(state, args[1]) && (isLikelyURL(args[2]) || strings.HasPrefix(args[2], "<"))
	case "remove", "rm", "show", "prune", "get-url":
		if len(args) < 2 {
			return false
		}
		return remoteExists(state, args[1])
	case "rename":
		if len(args) < 3 {
			return false
		}
		return remoteExists(state, args[1]) && !remoteExists(state, args[2])
	case "set-url":
		if len(args) < 3 {
			return false
		}
		return remoteExists(state, args[1]) && (isLikelyURL(args[2]) || strings.HasPrefix(args[2], "<"))
	default:
		return true
	}
}

func validateTagSuggestion(argv []string, state *status.GitState) bool {
	if len(argv) < 3 {
		return true
	}
	if target, ok := flagValue(argv[2:], "-d", "--delete"); ok {
		return tagExists(state, target)
	}
	positional := positionalArgs(argv[2:])
	if len(positional) == 0 {
		return true
	}
	return !tagExists(state, positional[0])
}

func validateRestoreSuggestion(argv []string, state *status.GitState) bool {
	args := argv[2:]
	if len(args) == 0 {
		return true
	}
	positional := positionalArgs(args)
	for _, arg := range positional {
		if arg == "." {
			return true
		}
		if !repoPathExists(arg) && !pathMentionedInWorkingTree(state, arg) {
			return false
		}
	}
	return true
}

func positionalArgs(args []string) []string {
	out := make([]string, 0, len(args))
	doubleDash := false
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if doubleDash {
			out = append(out, arg)
			continue
		}
		if arg == "--" {
			doubleDash = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			if optionConsumesValue(arg) && i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, arg)
	}
	return out
}

func flagValue(args []string, flags ...string) (string, bool) {
	match := make(map[string]struct{}, len(flags))
	for _, flag := range flags {
		match[flag] = struct{}{}
	}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if _, ok := match[arg]; ok {
			if i+1 < len(args) {
				return strings.TrimSpace(args[i+1]), true
			}
			return "", false
		}
	}
	return "", false
}

func argsAfterDoubleDash(args []string) []string {
	for i, arg := range args {
		if strings.TrimSpace(arg) == "--" {
			return append([]string(nil), args[i+1:]...)
		}
	}
	return nil
}

func containsToken(args []string, token string) bool {
	for _, arg := range args {
		if strings.TrimSpace(arg) == token {
			return true
		}
	}
	return false
}

func optionConsumesValue(flag string) bool {
	switch flag {
	case "-b", "-B", "-c", "-C", "-d", "-D", "-m", "--message", "--source", "--branch":
		return true
	default:
		return false
	}
}

func remoteExists(state *status.GitState, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || state == nil {
		return false
	}
	for _, remote := range state.Remotes {
		if strings.EqualFold(remote, name) {
			return true
		}
	}
	for _, remote := range state.RemoteInfos {
		if strings.EqualFold(remote.Name, name) {
			return true
		}
	}
	return false
}

func localBranchExists(state *status.GitState, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" || state == nil {
		return false
	}
	if strings.EqualFold(state.LocalBranch.Name, branch) {
		return true
	}
	for _, item := range state.LocalBranches {
		if strings.EqualFold(strings.TrimSpace(item), branch) {
			return true
		}
	}
	return false
}

func remoteBranchExists(state *status.GitState, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" || state == nil {
		return false
	}
	for _, item := range state.RemoteBranches {
		candidate := strings.TrimSpace(item)
		if strings.EqualFold(candidate, branch) {
			return true
		}
		if idx := strings.LastIndex(candidate, "/"); idx > 0 && strings.EqualFold(candidate[idx+1:], branch) {
			return true
		}
	}
	return false
}

func branchExists(state *status.GitState, branch string) bool {
	return localBranchExists(state, branch) || remoteBranchExists(state, branch)
}

func tagExists(state *status.GitState, tag string) bool {
	tag = strings.TrimSpace(tag)
	if tag == "" || state == nil {
		return false
	}
	for _, item := range state.Tags {
		if strings.EqualFold(strings.TrimSpace(item), tag) {
			return true
		}
	}
	return false
}

func currentBranch(state *status.GitState) string {
	if state == nil {
		return ""
	}
	return strings.TrimSpace(state.LocalBranch.Name)
}

func repoPathExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || strings.Contains(path, "\x00") {
		return false
	}
	if strings.HasPrefix(path, "<") && strings.HasSuffix(path, ">") {
		return false
	}
	clean := filepath.Clean(path)
	_, err := os.Stat(clean)
	return err == nil
}

func pathMentionedInWorkingTree(state *status.GitState, path string) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || state == nil {
		return false
	}
	for _, item := range state.WorkingTree {
		if filepath.Clean(item.Path) == path {
			return true
		}
	}
	for _, item := range state.StagingArea {
		if filepath.Clean(item.Path) == path {
			return true
		}
	}
	return false
}

func isLikelyURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "http://") ||
		strings.HasPrefix(value, "https://") ||
		strings.HasPrefix(value, "ssh://") ||
		(strings.Contains(value, "@") && strings.Contains(value, ":"))
}

func looksLikeCommitish(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	switch value {
	case "HEAD", "HEAD~1", "HEAD^":
		return true
	}
	if strings.HasPrefix(value, "HEAD~") || strings.HasPrefix(value, "HEAD^") {
		return true
	}
	if len(value) >= 7 && len(value) <= 40 {
		for _, r := range value {
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
				return false
			}
		}
		return true
	}
	return false
}
