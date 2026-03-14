package contributing

import "testing"

func TestParseContributing(t *testing.T) {
	spec := Parse("We use Conventional Commits and DCO. Branches follow feature/ and bugfix/.")
	if spec.CommitConvention != "conventional" || !spec.DCORequired || spec.BranchNaming != "gitflow" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
}
