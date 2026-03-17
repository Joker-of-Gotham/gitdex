package budget

import "testing"

func BenchmarkEstimateTokens(b *testing.B) {
	text := "GitDex context budget benchmark: 这是一个用于估算 token 的混合文本。"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = EstimateTokens(text)
	}
}

func BenchmarkCompressGitContent(b *testing.B) {
	content := `current_branch: main
## Local Branches
* main -> origin/main [ahead 2]
## Working Tree Changes
M internal/executor/runner.go
## Staging Area
M internal/llm/budget/budget.go
## Recent Reflog
...
## Commit Summary
commit_frequency: high
`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = CompressGitContent(content, 64)
	}
}
