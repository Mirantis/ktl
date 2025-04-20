package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Mirantis/rekustomize/pkg/cmd"
)

func main() {
	root := cmd.NewRootCommand()

	slog.SetLogLoggerLevel(slog.LevelInfo)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
