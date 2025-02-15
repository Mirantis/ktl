package filters

import (
	"strings"

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

type ValueMatcher struct {
	Value string
}

func (m *ValueMatcher) filterNilOrEmpty(input *yaml.RNode) (*yaml.RNode, error) {
	if m.Value == "" {
		return input, nil
	}
	return nil, nil
}

func (m *ValueMatcher) filterScalar(input *yaml.RNode) (*yaml.RNode, error) {
	if m.Value == input.YNode().Value {
		return input, nil
	}
	return nil, nil
}

func (m *ValueMatcher) Filter(input *yaml.RNode) (*yaml.RNode, error) {
	if input.IsNilOrEmpty() {
		return m.filterNilOrEmpty(input)
	}

	if "" == m.Value {
		return nil, nil
	}

	if input.YNode().Kind == yaml.ScalarNode {
		return m.filterScalar(input)
	}

	value, err := yaml.String(input.YNode(), yaml.Flow)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(value) == m.Value {
		return input, nil
	}

	return nil, nil
}
