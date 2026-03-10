package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileOperation represents a file system operation.
type FileOperation struct {
	Type    OpType
	Path    string
	Content string // for Create/Update
	Backup  bool   // create backup before modify/delete
}

type OpType string

const (
	OpRead   OpType = "read"
	OpCreate OpType = "create"
	OpUpdate OpType = "update"
	OpDelete OpType = "delete"
	OpExists OpType = "exists"
)

const repositoryFileMode = 0o644

// FileResult contains the result of a file operation.
type FileResult struct {
	Success    bool
	Path       string
	Content    string // for Read
	Exists     bool   // for Exists
	BackupPath string // if backup was created
	Error      error
}

// Executor performs file system operations.
type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

// Read reads a file's content.
func (e *Executor) Read(ctx context.Context, path string) (*FileResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return &FileResult{Success: false, Path: path, Error: err}, err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return &FileResult{Success: false, Path: absPath, Error: err}, err
	}
	return &FileResult{
		Success: true,
		Path:    absPath,
		Content: string(data),
	}, nil
}

// Create creates a new file with the given content.
func (e *Executor) Create(ctx context.Context, path string, content string) (*FileResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return &FileResult{Success: false, Path: path, Error: err}, err
	}
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &FileResult{Success: false, Path: absPath, Error: err}, err
	}
	if err := writeRepoFile(absPath, []byte(content)); err != nil {
		return &FileResult{Success: false, Path: absPath, Error: err}, err
	}
	return &FileResult{
		Success: true,
		Path:    absPath,
		Content: content,
	}, nil
}

// Update updates an existing file, optionally creating a backup.
func (e *Executor) Update(ctx context.Context, path string, content string, backup bool) (*FileResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return &FileResult{Success: false, Path: path, Error: err}, err
	}
	var backupPath string
	if backup {
		backupPath = absPath + ".bak"
		if _, err := os.Stat(absPath); err == nil {
			data, err := os.ReadFile(absPath)
			if err == nil {
				_ = writeRepoFile(backupPath, data)
			}
		}
	}
	if err := writeRepoFile(absPath, []byte(content)); err != nil {
		return &FileResult{Success: false, Path: absPath, Error: err}, err
	}
	return &FileResult{
		Success:    true,
		Path:       absPath,
		Content:    content,
		BackupPath: backupPath,
	}, nil
}

// Delete deletes a file, optionally creating a backup.
func (e *Executor) Delete(ctx context.Context, path string, backup bool) (*FileResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return &FileResult{Success: false, Path: path, Error: err}, err
	}
	var backupPath string
	if backup {
		backupPath = absPath + ".bak"
		if data, err := os.ReadFile(absPath); err == nil {
			_ = writeRepoFile(backupPath, data)
		}
	}
	if err := os.Remove(absPath); err != nil {
		return &FileResult{Success: false, Path: absPath, Error: err}, err
	}
	return &FileResult{
		Success:    true,
		Path:       absPath,
		BackupPath: backupPath,
	}, nil
}

// Exists checks if a file exists.
func (e *Executor) Exists(ctx context.Context, path string) (*FileResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return &FileResult{Success: false, Path: path, Error: err}, nil
	}
	_, err = os.Stat(absPath)
	exists := err == nil
	return &FileResult{
		Success: true,
		Path:    absPath,
		Exists:  exists,
	}, nil
}

// Append appends content to a file.
func (e *Executor) Append(ctx context.Context, path string, content string) (*FileResult, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return &FileResult{Success: false, Path: path, Error: err}, err
	}
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &FileResult{Success: false, Path: absPath, Error: err}, err
	}
	var existing string
	if data, err := os.ReadFile(absPath); err == nil {
		existing = string(data)
	}
	newContent := existing + content
	if err := writeRepoFile(absPath, []byte(newContent)); err != nil {
		return &FileResult{Success: false, Path: absPath, Error: err}, err
	}
	return &FileResult{
		Success: true,
		Path:    absPath,
		Content: newContent,
	}, nil
}

// ReadLines reads a file and returns its lines.
func (e *Executor) ReadLines(ctx context.Context, path string) ([]string, error) {
	result, err := e.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, result.Error
	}
	return strings.Split(strings.ReplaceAll(result.Content, "\r\n", "\n"), "\n"), nil
}

// UpdateLines updates specific lines in a file.
func (e *Executor) UpdateLines(ctx context.Context, path string, lineStart, lineEnd int, newLines []string, backup bool) (*FileResult, error) {
	lines, err := e.ReadLines(ctx, path)
	if err != nil {
		return nil, err
	}
	if lineStart < 0 || lineStart > len(lines) {
		return nil, fmt.Errorf("invalid line start: %d", lineStart)
	}
	if lineEnd < lineStart || lineEnd > len(lines) {
		return nil, fmt.Errorf("invalid line end: %d", lineEnd)
	}
	newContent := append(lines[:lineStart], append(newLines, lines[lineEnd:]...)...)
	content := strings.Join(newContent, "\n")
	return e.Update(ctx, path, content, backup)
}

// FindInFile searches for a pattern in a file and returns matching line numbers.
func (e *Executor) FindInFile(ctx context.Context, path string, pattern string) ([]int, error) {
	lines, err := e.ReadLines(ctx, path)
	if err != nil {
		return nil, err
	}
	var matches []int
	for i, line := range lines {
		if strings.Contains(line, pattern) {
			matches = append(matches, i+1) // 1-indexed
		}
	}
	return matches, nil
}

func writeRepoFile(path string, data []byte) error {
	// #nosec G306 -- repository files created by gitdex are expected to be readable by normal tooling.
	return os.WriteFile(path, data, repositoryFileMode)
}
