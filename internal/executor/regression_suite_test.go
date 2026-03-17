package executor

import (
	"testing"
)

func TestRegression_WhitespacePipeline_SingleTrailingNewline(t *testing.T) {
	in := "line 1  \r\nline 2\t \rline 3\n\n"
	got := stripTrailingWhitespace(in)
	want := "line 1\nline 2\nline 3\n"
	if got != want {
		t.Fatalf("unexpected sanitized content:\nwant=%q\ngot =%q", want, got)
	}
}

func TestRegression_GH404_DiagnosisIncluded(t *testing.T) {
	stderr := "gh: HTTP 404: Not Found (https://api.github.com/...)"
	got := classifyGHError(stderr, "gh release view v1.0.0")
	if !contains(got, "[GITDEX DIAGNOSIS]") {
		t.Fatalf("expected diagnosis marker in output: %q", got)
	}
	if !contains(got, "HTTP 404") {
		t.Fatalf("expected HTTP 404 hint in diagnosis: %q", got)
	}
}

