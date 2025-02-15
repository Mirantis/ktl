package filters

import (
	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

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
	mm := filters.MatchModifyFilter{
		MatchFilters: []yaml.YFilters{{yaml.YFilter{Filter: match}}},
	}

	if len(filter.Fields) == 0 {
		mm.ModifyFilters = yaml.YFilters{{Filter: &ValueSetter{}}}
	}

	for _, path := range filter.Fields {
		prefix, field := path[:len(path)-1], path[len(path)-1]
		tee := yaml.Tee(
			&yaml.PathMatcher{Path: prefix},
			&ForEach{Filters: yaml.YFilters{{Filter: yaml.Clear(field)}}},
		)
		mm.ModifyFilters = append(mm.ModifyFilters, yaml.YFilter{Filter: tee})
	}

	if _, err := mm.Filter(input); err != nil {
		return nil, err
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
