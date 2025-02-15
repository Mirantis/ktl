package filters

import "sigs.k8s.io/kustomize/kyaml/yaml"

type ForEach struct {
	Filters []yaml.Filter
}

func (f *ForEach) Filter(input *yaml.RNode) (*yaml.RNode, error) {
	nodes, err := input.Elements()
	if err != nil {
		return nil, err
	}
	for _, rn := range nodes {
		if err := rn.PipeE(f.Filters...); err != nil {
			return nil, err
		}
	}
	return input, nil
}
