package filters

import (
	"fmt"

	"github.com/Mirantis/ktl/pkg/resource"
	"github.com/Mirantis/ktl/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const skipAnnotation = "x-ktl-skip"

//nolint:gochecknoinits
func init() {
	filters.Filters["SkipFilter"] = func() kio.Filter { return &SkipFilter{} }
}

type SkipFilter struct {
	Kind      string            `yaml:"kind"`
	Resources []*types.Selector `yaml:"resources"`
	Except    []*types.Selector `yaml:"except"`
	Fields    []resource.Query  `yaml:"fields"`
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
		matchModify.ModifyFilters = yaml.YFilters{{
			Filter: yaml.SetAnnotation(skipAnnotation, "true"),
		}}
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

	for _, rnode := range input {
		if rnode.IsNilOrEmpty() {
			continue
		}

		shouldSkip, _ := rnode.MatchesAnnotationSelector(skipAnnotation + "=true")
		if shouldSkip {
			continue
		}

		output = append(output, rnode)
	}

	return output, nil
}
