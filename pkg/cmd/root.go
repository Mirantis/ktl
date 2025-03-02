package cmd

import "github.com/spf13/cobra"

func RootCommand() *cobra.Command {
	root := &cobra.Command{
		Use: "rekustomize",
	}
	root.AddCommand(exportCommand())

	return root
}
