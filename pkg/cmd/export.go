package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func exportCommand() *cobra.Command {
	opts := &exportOpts{}
	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented")
		},
	}
	export.Flags().StringSliceVarP(&opts.nsFilter, "namespace-filter", "n", nil, "TODO: usage")
	export.Flags().StringVarP(&opts.server, "server", "s", "", "TODO: usage")
	return export
}

type exportOpts struct {
	nsFilter []string
	server   string
}
