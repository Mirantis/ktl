package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mirantis/ktl/pkg/fsutil"
	"github.com/Mirantis/ktl/pkg/kubectl"
	"github.com/Mirantis/ktl/pkg/runner"
	"github.com/Mirantis/ktl/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func newRunCommand() *cobra.Command {
	export := &cobra.Command{
		Use:   "run FILENAME",
		Short: "execute the pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:revive
			fileName := args[0]
			workDir := filepath.Dir(fileName)
			fileSys := fsutil.Stdio(
				fsutil.Sub(filesys.MakeFsOnDisk(), workDir),
				cmd.InOrStdin(), cmd.OutOrStdout(),
			)
			env := &types.Env{
				WorkDir: workDir,
				FileSys: fileSys,
				Cmd:     kubectl.New(),
			}

			pipelineBytes, err := os.ReadFile(fileName)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("unable to read %s: %w", fileName, err)
			}

			pipeline := &runner.Pipeline{}
			if err := yaml.Unmarshal(pipelineBytes, pipeline); err != nil {
				return fmt.Errorf("unable to parse %s: %w", fileName, err)
			}

			return pipeline.Run(env)
		},
	}

	return export
}
