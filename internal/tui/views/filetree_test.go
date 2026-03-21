package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestFilesView_ID(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	if v.ID() != ViewFiles {
		t.Errorf("ID() = %q, want %q", v.ID(), ViewFiles)
	}
	if v.Title() != "Files" {
		t.Errorf("Title() = %q, want Files", v.Title())
	}
}

func TestFilesView_TreeNavigation(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 40)

	root := &FileEntry{
		Name: ".", IsDir: true, Expanded: true,
		Children: []*FileEntry{
			{Name: "src", IsDir: true, Children: []*FileEntry{
				{Name: "main.go", Path: "src/main.go"},
			}},
			{Name: "README.md", Path: "README.md"},
		},
	}
	v.SetTree(root)

	if len(v.flatList) != 3 { // root + src + README (src not expanded)
		t.Errorf("flatList = %d, want 3", len(v.flatList))
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if v.cursor != 1 {
		t.Errorf("cursor = %d, want 1", v.cursor)
	}

	// Expand src directory
	v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !v.flatList[1].Expanded {
		t.Error("Enter on dir should toggle expanded")
	}
	if len(v.flatList) != 4 { // root + src + main.go + README
		t.Errorf("flatList after expand = %d, want 4", len(v.flatList))
	}
}

func TestFilesView_CodeMode(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 40)

	v.Update(FileContentMsg{
		Path:    "test.go",
		Content: "package main\n\nfunc main() {\n}\n",
	})

	if v.mode != modeCode {
		t.Error("FileContentMsg should switch to code mode")
	}
	if v.codePath != "test.go" {
		t.Errorf("codePath = %q, want test.go", v.codePath)
	}

	output := v.Render()
	if !strings.Contains(output, "test.go") {
		t.Error("code view should show file path")
	}
	plain := stripANSI(output)
	if !strings.Contains(plain, "package") {
		t.Error("code view should show file content")
	}
}

func TestFilesView_DiffMode(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 40)

	v.Update(FileDiffMsg{
		Diff: "--- a/test.go\n+++ b/test.go\n@@ -1 +1 @@\n-old\n+new\n",
	})

	if v.mode != modeDiff {
		t.Error("FileDiffMsg should switch to diff mode")
	}

	output := v.Render()
	if !strings.Contains(output, "Diff") {
		t.Error("diff view should show title")
	}
}

func TestFilesView_EscBackToTree(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 40)

	v.mode = modeCode
	v.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v.mode != modeTree {
		t.Error("Esc should return to tree mode")
	}

	v.mode = modeDiff
	v.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v.mode != modeTree {
		t.Error("Esc should return to tree mode from diff")
	}

	v.mode = modeEdit
	v.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v.mode != modeCode {
		t.Error("Esc should return to code mode from edit")
	}
}

func TestFilesView_RenderEmpty(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 40)

	output := v.Render()
	if !strings.Contains(output, "No files loaded") {
		t.Error("empty tree should show message")
	}
}

func TestBuildFileTree(t *testing.T) {
	entries := []string{
		"cmd/main.go",
		"internal/app/app.go",
		"internal/app/app_test.go",
		"go.mod",
		"README.md",
	}
	root := BuildFileTree(entries)

	if root.Name != "." {
		t.Errorf("root name = %q, want '.'", root.Name)
	}
	if !root.IsDir {
		t.Error("root should be a directory")
	}
	if len(root.Children) != 4 { // cmd, internal, go.mod, README.md
		t.Errorf("root children = %d, want 4", len(root.Children))
	}
}

func TestBuildFileTree_NestedDirs(t *testing.T) {
	entries := []string{
		"a/b/c/file.go",
	}
	root := BuildFileTree(entries)

	if len(root.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(root.Children))
	}
	aDir := root.Children[0]
	if aDir.Name != "a" || !aDir.IsDir {
		t.Error("first child should be 'a' dir")
	}
	if len(aDir.Children) != 1 || aDir.Children[0].Name != "b" {
		t.Error("'a' should contain 'b'")
	}
}

func TestFileIconForName(t *testing.T) {
	tests := map[string]bool{
		"main.go":     true,
		"config.yaml": true,
		"README.md":   true,
		"data.json":   true,
		"unknown.xyz": true,
	}
	for name := range tests {
		icon := fileIconForName(name)
		if icon == "" {
			t.Errorf("fileIconForName(%q) returned empty", name)
		}
	}
}

func TestFilesView_CodeScroll(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 10)

	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "line content")
	}
	v.Update(FileContentMsg{Path: "big.go", Content: strings.Join(lines, "\n")})

	if v.mode != modeCode {
		t.Error("should be in code mode after FileContentMsg")
	}

	out1 := v.Render()
	v.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	out2 := v.Render()
	_ = out1
	_ = out2
}

func TestFilesView_EditModeAndSaveResult(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 20)
	v.SetEditable(true)

	_, cmd := v.Update(FileEditMsg{
		Path:    "main.go",
		Content: "package main\n",
	})
	if v.mode != modeEdit {
		t.Fatal("FileEditMsg should switch to edit mode")
	}
	if cmd == nil {
		t.Fatal("FileEditMsg should return editor focus/blink command")
	}

	v.editor.SetValue("package main\n\nfunc main() {}\n")
	_, saveCmd := v.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if saveCmd == nil {
		t.Fatal("Ctrl+S should request file save")
	}
	msg := saveCmd()
	saveReq, ok := msg.(RequestFileSaveMsg)
	if !ok {
		t.Fatalf("save message = %T, want RequestFileSaveMsg", msg)
	}
	if !strings.Contains(saveReq.Content, "func main") {
		t.Fatalf("save content = %q", saveReq.Content)
	}

	v.Update(FileSavedMsg{Path: "main.go", Content: saveReq.Content})
	if v.mode != modeCode {
		t.Fatal("successful save should return to code mode")
	}
	if !strings.Contains(v.codeContent, "func main") {
		t.Fatal("saved content should update code content")
	}
}

func TestFilesView_ModalEditorFlow(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 20)
	v.SetEditable(true)

	v.Update(FileEditMsg{
		Path:    "main.go",
		Content: "package main\n",
	})
	if v.editorMode != editorNormal {
		t.Fatalf("editorMode = %v, want normal", v.editorMode)
	}

	v.Update(tea.KeyPressMsg{Code: 'i'})
	if v.editorMode != editorInsert {
		t.Fatal("i should switch to insert mode")
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if v.editorMode != editorNormal {
		t.Fatal("Esc should return to normal mode from insert")
	}

	v.Update(tea.KeyPressMsg{Code: 'l'})
	if v.editorMode != editorNormal || v.mode != modeEdit {
		t.Fatal("normal mode movement should keep editor open in normal mode")
	}
}

func TestFilesView_RemoteCloneRequest(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 20)
	v.SetRepository(RepoListItem{FullName: "owner/repo", Name: "repo"})

	_, cmd := v.Update(tea.KeyPressMsg{Code: 'c'})
	if cmd == nil {
		t.Fatal("c in remote-only mode should request a local clone")
	}
	msg := cmd()
	req, ok := msg.(CloneRepoRequestMsg)
	if !ok {
		t.Fatalf("clone request msg = %T", msg)
	}
	if req.Repo.FullName != "owner/repo" {
		t.Fatalf("repo = %q, want owner/repo", req.Repo.FullName)
	}
}

func TestFilesView_CreateFilePrompt(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 20)
	v.SetEditable(true)
	v.SetTree(&FileEntry{Name: ".", IsDir: true, Expanded: true})

	_, cmd := v.Update(tea.KeyPressMsg{Code: 'n'})
	if cmd == nil {
		t.Fatal("n should open a create file prompt")
	}
	if v.promptKind != filePromptCreateFile {
		t.Fatalf("promptKind = %v, want create file", v.promptKind)
	}

	v.prompt.SetValue("notes/new.txt")
	_, submitCmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if submitCmd == nil {
		t.Fatal("enter should submit the file operation prompt")
	}
	msg := submitCmd()
	req, ok := msg.(RequestFileOpMsg)
	if !ok {
		t.Fatalf("request msg = %T", msg)
	}
	if req.Kind != FileOpCreateFile {
		t.Fatalf("kind = %q, want %q", req.Kind, FileOpCreateFile)
	}
	if req.Target != "notes/new.txt" {
		t.Fatalf("target = %q, want notes/new.txt", req.Target)
	}
}

func TestFilesView_DeletePromptRequiresLiteralDelete(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewFilesView(&th)
	v.SetSize(120, 20)
	v.SetEditable(true)
	v.SetTree(&FileEntry{
		Name: ".", IsDir: true, Expanded: true,
		Children: []*FileEntry{{Name: "main.go", Path: "main.go"}},
	})

	v.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := v.Update(tea.KeyPressMsg{Code: 'x'})
	if cmd == nil {
		t.Fatal("x should open a delete confirmation prompt")
	}
	if v.promptKind != filePromptDelete {
		t.Fatalf("promptKind = %v, want delete", v.promptKind)
	}

	v.prompt.SetValue("delete")
	_, deleteCmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if deleteCmd == nil {
		t.Fatal("delete confirmation should submit an operation")
	}
	msg := deleteCmd()
	req, ok := msg.(RequestFileOpMsg)
	if !ok {
		t.Fatalf("request msg = %T", msg)
	}
	if req.Kind != FileOpDelete || req.Path != "main.go" {
		t.Fatalf("request = %#v, want delete main.go", req)
	}
}
