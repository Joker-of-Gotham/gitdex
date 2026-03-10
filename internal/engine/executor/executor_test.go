package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCLI struct {
	execFunc func(ctx context.Context, args ...string) (string, string, error)
}

func (m *mockCLI) Exec(ctx context.Context, args ...string) (string, string, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, args...)
	}
	return "", "", nil
}

func (m *mockCLI) ExecStream(ctx context.Context, args ...string) (<-chan string, error) {
	return nil, nil
}

func (m *mockCLI) Version() (string, error) {
	return "", nil
}

var _ cli.GitCLI = (*mockCLI)(nil)

func TestNewCommandExecutor(t *testing.T) {
	m := &mockCLI{}
	e := NewCommandExecutor(m)
	require.NotNil(t, e)
}

func TestCommit_Success(t *testing.T) {
	m := &mockCLI{
		execFunc: func(ctx context.Context, args ...string) (string, string, error) {
			return "output", "", nil
		},
	}
	e := NewCommandExecutor(m)
	res, err := e.Commit(context.Background(), "feat: add x")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Success)
	assert.Equal(t, []string{"git", "commit", "-m", "feat: add x"}, res.Command)
	assert.Equal(t, "output", res.Stdout)
}

func TestCommit_Error(t *testing.T) {
	wantErr := errors.New("git failed")
	m := &mockCLI{
		execFunc: func(ctx context.Context, args ...string) (string, string, error) {
			return "", "error", wantErr
		},
	}
	e := NewCommandExecutor(m)
	res, err := e.Commit(context.Background(), "x")
	assert.ErrorIs(t, err, wantErr)
	require.NotNil(t, res)
	assert.False(t, res.Success)
}

func TestPush_Success(t *testing.T) {
	m := &mockCLI{
		execFunc: func(ctx context.Context, args ...string) (string, string, error) {
			return "pushed", "", nil
		},
	}
	e := NewCommandExecutor(m)
	res, err := e.Push(context.Background())
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Success)
	assert.Equal(t, []string{"git", "push"}, res.Command)
}

func TestStageAll_Success(t *testing.T) {
	m := &mockCLI{
		execFunc: func(ctx context.Context, args ...string) (string, string, error) {
			assert.Equal(t, []string{"add", "."}, args)
			return "", "", nil
		},
	}
	e := NewCommandExecutor(m)
	res, err := e.StageAll(context.Background())
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Success)
	assert.Equal(t, []string{"git", "add", "."}, res.Command)
}

func TestStageFiles_Success(t *testing.T) {
	m := &mockCLI{
		execFunc: func(ctx context.Context, args ...string) (string, string, error) {
			assert.Equal(t, []string{"add", "a.go", "b.go"}, args)
			return "", "", nil
		},
	}
	e := NewCommandExecutor(m)
	res, err := e.StageFiles(context.Background(), []string{"a.go", "b.go"})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Success)
	assert.Equal(t, []string{"git", "add", "a.go", "b.go"}, res.Command)
}

func TestStageFiles_Empty(t *testing.T) {
	e := NewCommandExecutor(&mockCLI{})
	res, err := e.StageFiles(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Success)
}

func TestExecute_MultipleCommands(t *testing.T) {
	var callCount int
	m := &mockCLI{
		execFunc: func(ctx context.Context, args ...string) (string, string, error) {
			callCount++
			return "ok", "", nil
		},
	}
	e := NewCommandExecutor(m)
	res, err := e.Execute(context.Background(), []string{"status", "add ."})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.Success)
	assert.Equal(t, 2, callCount)
}
