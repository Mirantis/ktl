package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/fsutil"
	"github.com/Mirantis/ktl/pkg/kubectl"
	"github.com/Mirantis/ktl/pkg/runner"
	"github.com/Mirantis/ktl/pkg/types"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
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

			pipelineJSON, err := yaml.YAMLToJSON(pipelineBytes)
			if err != nil {
				return err
			}

			pipelineSpec := &apis.Pipeline{}

			if err := protojson.Unmarshal(pipelineJSON, pipelineSpec); err != nil {
				return err
			}

			pipeline, err := runner.NewPipeline(pipelineSpec)
			if err != nil {
				return err
			}

			return pipeline.Run(env)
		},
	}

	return export
}
