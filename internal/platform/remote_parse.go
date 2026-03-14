package platform

import "github.com/Joker-of-Gotham/gitdex/internal/git"

func GitHubOwnerRepoFromRemote(remoteURL string) (owner, repo string, err error) {
	return parseGitHubOwnerRepo(remoteURL)
}

func GitLabProjectPathFromRemote(remoteURL string) (string, error) {
	return parseGitLabProjectPath(remoteURL)
}

func BitbucketWorkspaceRepoFromRemote(remoteURL string) (workspace, repo string, err error) {
	return parseBitbucketWorkspaceRepo(remoteURL)
}

func PreferredRemoteURL(infos []git.RemoteInfo) string {
	if len(infos) == 0 {
		return ""
	}
	remoteURL := preferredRemoteURL(infos[0])
	for _, info := range infos {
		if info.Name == "origin" {
			return preferredRemoteURL(info)
		}
	}
	return remoteURL
}
