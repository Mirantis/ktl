package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mirantis/rekustomize/pkg/config"
	_ "github.com/Mirantis/rekustomize/pkg/filters" // register filters
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func exportCommand() *cobra.Command {
	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:revive
			env := &config.Env{
				WorkDir: args[0],
				FileSys: filesys.MakeFsOnDisk(),
				Cmd:     kubectl.New(),
			}

			cfgData, err := os.ReadFile(filepath.Join(env.WorkDir, "rekustomization.yaml"))
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("unable to read rekustomization.yaml: %w", err)
			}

			cfg := &config.Rekustomization{}
			if err := yaml.Unmarshal(cfgData, cfg); err != nil {
				return fmt.Errorf("unable to parse rekustomization.yaml: %w", err)
			}

			return cfg.Run(env)
		},
	}

	return export
}
