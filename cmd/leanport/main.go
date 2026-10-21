package main

import (
	"fmt"
	"os"

	"github.com/shrsv/LeanPort/internal/app"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app.Version = version
	app.Commit = commit
	app.Date = date
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "leanport:", err)
		if app.IsUsage(err) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
