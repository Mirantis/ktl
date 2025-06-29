package cmd

import (
	"fmt"
	"os"

	"github.com/Mirantis/ktl/pkg/apis"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

func loadPipelineSpec(path string) (*apis.Pipeline, error) {
	pipelineBytes, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to read %s: %w", path, err)
	}

	pipelineJSON, err := yaml.YAMLToJSON(pipelineBytes)
	if err != nil {
		return nil, err
	}

	pipelineSpec := &apis.Pipeline{}

	if err := protojson.Unmarshal(pipelineJSON, pipelineSpec); err != nil {
		return nil, err
	}

	return pipelineSpec, nil
}
