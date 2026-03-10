package main

import (
	"fmt"
	"os"

	"github.com/Joker-of-Gotham/gitdex/internal/app"
)

var version = "dev"

func main() {
	application := app.New(app.Config{
		Version: version,
	})
	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
