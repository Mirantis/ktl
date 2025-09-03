package output

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/types"
)

func newJSONOutput(spec *apis.JSONOutput) (*JSONOutput, error) {
	return &JSONOutput{spec}, nil
}

type JSONOutput struct {
	*apis.JSONOutput
}

func (out *JSONOutput) Store(env *types.Env, resources *types.ClusterResources) error {
	path := out.GetPath()
	if filepath.IsAbs(path) {
		return fmt.Errorf("invalid JSON output path: %w", errAbsPath)
	}

	body := []byte{}

	for clusterID, _ := range resources.Clusters.All() {
		for _, rnode := range resources.All(&clusterID) {
			if len(body) > 0 {
				return fmt.Errorf("JSON output only supports single-object results")
			}

			var err error
			body, err = json.MarshalIndent(rnode, "", "  ")
			if err != nil {
				return err
			}
		}
	}

	return env.FileSys.WriteFile(path, body) //nolint:wrapcheck
}
