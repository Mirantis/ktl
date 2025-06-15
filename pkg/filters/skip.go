package filters

import (
	"fmt"

	"github.com/Mirantis/ktl/pkg/apis"
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

func newSelectors(specs []*apis.ResourceSelector) []*types.Selector {
	result := []*types.Selector{}
	for _, rsSpec := range specs {
		rs := &types.Selector{}
		rs.Group = rsSpec.GetGroup()
		rs.Version = rsSpec.GetVersion()
		rs.Kind = rsSpec.GetKind()
		rs.Name = rsSpec.GetName()
		rs.Namespace = rsSpec.GetNamespace()
		rs.LabelSelector = rsSpec.GetLabelSelector()
		rs.AnnotationSelector = rsSpec.GetAnnotationSelector()
		result = append(result, rs)
	}

	return result
}

func newSkipFilter(spec *apis.SkipFilter) (*SkipFilter, error) {
	sf := &SkipFilter{}

	sf.Resources = newSelectors(spec.GetResources())
	sf.Except = newSelectors(spec.GetKeepResources())

	for _, fSpec := range spec.GetFields() {
		q := resource.Query{}
		err := q.UnmarshalYAML(yaml.NewStringRNode(fSpec).YNode())
		if err != nil {
			return nil, err
		}
		sf.Fields = append(sf.Fields, q)
	}

	return sf, nil
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
