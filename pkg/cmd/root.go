package cmd

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use: "ktl",
	}
	root.AddCommand(newRunCommand())
	root.AddCommand(newMCPCommand())
	root.AddCommand(newQueryCommand())

	return root
}
