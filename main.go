package main

import (
	"fmt"
	"os"

	"github.com/Mirantis/rekustomize/pkg/cmd"
)

func main() {
	root := cmd.RootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
