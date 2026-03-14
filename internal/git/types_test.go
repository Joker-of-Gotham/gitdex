package git

import (
	"errors"
	"testing"
)

func TestAppErrorUnwrap(t *testing.T) {
	root := errors.New("root")
	err := &AppError{Message: "wrapped", Cause: root}
	if !errors.Is(err, root) {
		t.Fatal("expected unwrap support")
	}
}
