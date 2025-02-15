package filters

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func init() {
	yaml.Filters["ValueSetter"] = func() yaml.Filter { return &ValueSetter{} }
}

type ValueSetter struct {
	Kind  string      `yaml:"kind"`
	Value *yaml.RNode `yaml:"value"`
}

func (filter *ValueSetter) Filter(input *yaml.RNode) (*yaml.RNode, error) {
	v := filter.Value
	if v == nil {
		v = yaml.MakeNullNode()
	}
	*input = *v.Copy()
	return input, nil
}
