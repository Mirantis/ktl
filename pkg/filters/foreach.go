package filters

import "sigs.k8s.io/kustomize/kyaml/yaml"

func init() {
	yaml.Filters["ForEach"] = func() yaml.Filter { return &ForEach{} }
}

type ForEach struct {
	Kind    string        `yaml:"kind"`
	Filters yaml.YFilters `yaml:"pipeline"`
}

func (f *ForEach) Filter(input *yaml.RNode) (*yaml.RNode, error) {
	nodes, err := input.Elements()
	if err != nil {
		return nil, err
	}
	for _, rn := range nodes {
		if err := rn.PipeE(f.Filters.Filters()...); err != nil {
			return nil, err
		}
	}
	return input, nil
}
