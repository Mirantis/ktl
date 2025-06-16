package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	var debug bool
	root := &cobra.Command{
		Use: "ktl",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				slog.SetLogLoggerLevel(slog.LevelInfo)
			}
		},
	}

	root.PersistentFlags().BoolVarP(
		&debug,
		"debug", "d",
		false,
		"enable debug logging",
	)

	root.AddCommand(newRunCommand())
	root.AddCommand(newMCPCommand())
	root.AddCommand(newQueryCommand())

	return root
}
