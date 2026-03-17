package executor

import "testing"

func BenchmarkParseCommand(b *testing.B) {
	cmd := `gh issue create --title "Stabilize parser on Windows path C:\work\repo" --body "Ensure no escaping bugs"`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = parseCommand(cmd)
	}
}

func BenchmarkStripTrailingWhitespace(b *testing.B) {
	input := "line 1   \r\nline 2\t\t\r\nline 3\u00A0\u00A0\r\n"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = stripTrailingWhitespace(input)
	}
}
