package filters

import (
	"fmt"

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
	if input == nil {
		return nil, fmt.Errorf("unable to set nil node")
	}
	v := filter.Value
	if v == nil {
		v = yaml.MakeNullNode()
	}
	*input = *v
	return input, nil
}
