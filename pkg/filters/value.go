package filters

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//nolint:gochecknoinits
func init() {
	yaml.Filters["ValueSetter"] = func() yaml.Filter { return &ValueSetter{} }
}

type ValueSetter struct {
	Kind  string      `yaml:"kind"`
	Value *yaml.RNode `yaml:"value"`
}

func (filter *ValueSetter) Filter(input *yaml.RNode) (*yaml.RNode, error) {
	if filter.Value == nil {
		input.SetYNode(nil)

		return input, nil
	}

	*input = *filter.Value.Copy()

	return input, nil
}

type ValueMatcher struct {
	Value string
}

func (m *ValueMatcher) filterNilOrEmpty(input *yaml.RNode) (*yaml.RNode, error) {
	if m.Value == "" {
		return input, nil
	}

	return nil, nil //nolint:nilnil
}

func (m *ValueMatcher) filterScalar(input *yaml.RNode) (*yaml.RNode, error) {
	if m.Value == input.YNode().Value {
		return input, nil
	}

	return nil, nil //nolint:nilnil
}

func (m *ValueMatcher) Filter(input *yaml.RNode) (*yaml.RNode, error) {
	if input.IsNilOrEmpty() {
		return m.filterNilOrEmpty(input)
	}

	if m.Value == "" {
		return nil, nil //nolint:nilnil
	}

	if input.YNode().Kind == yaml.ScalarNode {
		return m.filterScalar(input)
	}

	value, err := yaml.String(input.YNode(), yaml.Flow)
	if err != nil {
		return nil, fmt.Errorf("invalid yaml node: %w", err)
	}

	if strings.TrimSpace(value) == m.Value {
		return input, nil
	}

	return nil, nil //nolint:nilnil
}
