package main

import (
	"fmt"
	"os"

	"github.com/Mirantis/ktl/pkg/cmd"
	_ "github.com/Mirantis/ktl/pkg/filters" // register filters
)

func main() {
	root := cmd.NewRootCommand()

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
