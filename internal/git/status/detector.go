package status

import "strconv"

// DetectAnomalies checks for unexpected state changes between previous and current GitState.
func DetectAnomalies(prev, curr *GitState) []string {
	var anomalies []string
	if prev == nil || curr == nil {
		return anomalies
	}

	// Check if commits suddenly disappeared (HEAD moved backward unexpectedly)
	if prev.HeadRef != "" && curr.HeadRef != "" && prev.HeadRef != curr.HeadRef {
		// HEAD changed - could be intentional (new commit, checkout, reset) or anomaly.
		// We cannot tell from state alone if it was "commits disappeared" vs normal operation.
		// Caller may pass additional context. For now we note HEAD changed.
		// A more advanced check would compare commit ancestry (prev.HeadRef reachable from curr.HeadRef?).
		// Simplified: if we had a branch and now we're detached with different ref, flag it
		if prev.LocalBranch.Name != "" && curr.LocalBranch.IsDetached {
			anomalies = append(anomalies, "branch changed to detached HEAD; previous branch state may be lost")
		}
	}

	// Check if branch changed unexpectedly (e.g. from main to different branch without explicit checkout)
	if prev.LocalBranch.Name != "" && curr.LocalBranch.Name != "" &&
		prev.LocalBranch.Name != curr.LocalBranch.Name {
		anomalies = append(anomalies, "branch changed from "+prev.LocalBranch.Name+" to "+curr.LocalBranch.Name)
	}

	// Check if we were on a branch and are now detached
	if prev.LocalBranch.Name != "" && !prev.LocalBranch.IsDetached &&
		curr.LocalBranch.IsDetached {
		anomalies = append(anomalies, "detached HEAD: previous branch "+prev.LocalBranch.Name+" may need to be checked out to restore")
	}

	// Check if stash entries dropped significantly (potential stash drop/clear)
	prevStash := len(prev.StashStack)
	currStash := len(curr.StashStack)
	if prevStash > 0 && currStash < prevStash {
		anomalies = append(anomalies, "stash entries decreased from "+strconv.Itoa(prevStash)+" to "+strconv.Itoa(currStash))
	}

	return anomalies
}
