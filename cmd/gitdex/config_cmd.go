package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
)

func runConfigCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: gitdex config <lint|explain|source|schema>")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	trace := config.LastLoadTrace()

	switch args[0] {
	case "lint":
		result := config.Lint(cfg)
		if result.Valid {
			fmt.Println("OK: config is valid")
		} else {
			fmt.Println("FAIL: config is invalid")
		}
		for _, w := range result.Warnings {
			fmt.Printf("- %s\n", w)
		}
		if !result.Valid {
			return fmt.Errorf("config lint failed")
		}
		return nil
	case "explain":
		fmt.Print(config.Explain(cfg, trace))
		return nil
	case "source":
		out, err := config.SourceTraceJSON(trace)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	case "schema":
		path := filepath.FromSlash("configs/schema.gitdexrc.json")
		abs, _ := filepath.Abs(path)
		fmt.Println(abs)
		return nil
	default:
		return fmt.Errorf("unknown config subcommand %q", args[0])
	}
}

func maybeHandleConfigCommand() error {
	if len(os.Args) < 2 || os.Args[1] != "config" {
		return nil
	}
	return runConfigCommand(os.Args[2:])
}

