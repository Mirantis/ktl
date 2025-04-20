package cmd

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use: "rekustomize",
	}
	root.AddCommand(newRunCommand())

	return root
}
