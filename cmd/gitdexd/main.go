package main

import (
	"fmt"
	"os"

	"github.com/your-org/gitdex/internal/cli/command"
)

func main() {
	if err := command.NewDaemonBinaryRootCommand().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
