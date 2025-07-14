package cmd

import (
	"path/filepath"

	"github.com/Mirantis/ktl/pkg/fsutil"
	"github.com/Mirantis/ktl/pkg/kubectl"
	"github.com/Mirantis/ktl/pkg/runner"
	"github.com/Mirantis/ktl/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
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

			pipelineSpec, err := loadPipelineSpec(fileName)
			if err != nil {
				return err
			}

			pipeline, err := runner.NewPipeline(pipelineSpec, nil)
			if err != nil {
				return err
			}

			return pipeline.Run(env)
		},
	}

	return export
}
