package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/Mirantis/rekustomize/pkg/filters" // register filters
	"github.com/Mirantis/rekustomize/pkg/fsutil"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/runner"
	"github.com/Mirantis/rekustomize/pkg/types"
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
			env := &types.Env{
				WorkDir: workDir,
				FileSys: fsutil.Sub(filesys.MakeFsOnDisk(), workDir),
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
