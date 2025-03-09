package filters

import (
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type ForEach struct {
	Filters []yaml.Filter
}

func (f *ForEach) Filter(input *yaml.RNode) (*yaml.RNode, error) {
	nodes, err := input.Elements()
	if err != nil {
		return nil, fmt.Errorf("invalid yaml node: %w", err)
	}

	for _, rn := range nodes {
		if err := rn.PipeE(f.Filters...); err != nil {
			return nil, fmt.Errorf("unable to apply yaml filter: %w", err)
		}
	}

	return input, nil
}
