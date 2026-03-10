package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripThinking_NonThinkingModel(t *testing.T) {
	input := `[{"action":"Stage files","command":"git add .","reason":"Untracked files","risk":"safe"}]`
	assert.Equal(t, input, StripThinking(input))
}

func TestStripThinking_ThinkingModel_Qwen3(t *testing.T) {
	input := `<think>
The user has untracked files. I should suggest git add.
Let me analyze the repository state...
</think>
[{"action":"Stage files","command":"git add .","reason":"Untracked files","risk":"safe"}]`
	expected := `[{"action":"Stage files","command":"git add .","reason":"Untracked files","risk":"safe"}]`
	assert.Equal(t, expected, StripThinking(input))
}

func TestStripThinking_ThinkingModel_MultipleBlocks(t *testing.T) {
	input := `<think>first thought</think>hello <think>second thought</think>world`
	assert.Equal(t, "hello world", StripThinking(input))
}

func TestStripThinking_EmptyThink(t *testing.T) {
	input := `<think></think>git add .`
	assert.Equal(t, "git add .", StripThinking(input))
}

func TestStripThinking_NoTags(t *testing.T) {
	assert.Equal(t, "just plain text", StripThinking("just plain text"))
}

func TestStripThinking_OnlyThink(t *testing.T) {
	input := `<think>all thinking no output</think>`
	assert.Equal(t, "", StripThinking(input))
}

func TestStripThinking_CommitMessage(t *testing.T) {
	input := `<think>
I need to generate a commit message for the staged changes.
The user has modified main.go and added a new test file.
</think>
Add unit tests and fix main entry point`
	assert.Equal(t, "Add unit tests and fix main entry point", StripThinking(input))
}
