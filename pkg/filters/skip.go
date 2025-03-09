package filters

import (
	"fmt"

	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//nolint:gochecknoinits
func init() {
	filters.Filters["SkipFilter"] = func() kio.Filter { return &SkipFilter{} }
}

type SkipFilter struct {
	Kind      string            `yaml:"kind"`
	Resources []*types.Selector `yaml:"resources"`
	Except    []*types.Selector `yaml:"except"`
	Fields    []types.NodePath  `yaml:"fields"`
}

func (filter *SkipFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	match := &ResourceMatcher{
		Resources: filter.Resources,
		Except:    filter.Except,
	}
	matchModify := filters.MatchModifyFilter{
		MatchFilters: []yaml.YFilters{{yaml.YFilter{Filter: match}}},
	}

	if len(filter.Fields) == 0 {
		matchModify.ModifyFilters = yaml.YFilters{{Filter: &ValueSetter{}}}
	}

	for _, path := range filter.Fields {
		clearAll, err := ClearAll(path)
		if err != nil {
			return nil, err
		}

		yf := yaml.YFilter{Filter: clearAll}
		matchModify.ModifyFilters = append(matchModify.ModifyFilters, yf)
	}

	if _, err := matchModify.Filter(input); err != nil {
		return nil, fmt.Errorf("unable to apply filter: %w", err)
	}

	output := []*yaml.RNode{}

	for _, rn := range input {
		if rn.IsNilOrEmpty() {
			continue
		}

		output = append(output, rn)
	}

	return output, nil
}
