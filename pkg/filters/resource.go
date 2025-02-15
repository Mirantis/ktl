package filters

import (
	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func init() {
	yaml.Filters["ResourceMatcher"] = func() yaml.Filter { return &ResourceMatcher{} }
}

type ResourceMatcher struct {
	Kind      string            `yaml:"kind"`
	Resources []*types.Selector `yaml:"resources"`
	Except    []*types.Selector `yaml:"except"`
}

func (m *ResourceMatcher) Filter(input *yaml.RNode) (*yaml.RNode, error) {
	if len(m.Resources) > 0 && !matchSelectors(input, m.Resources) {
		return nil, nil
	}
	if len(m.Except) > 0 && matchSelectors(input, m.Except) {
		return nil, nil
	}
	return input, nil
}

func matchSelectors(rn *yaml.RNode, selectors []*types.Selector) bool {
	id := resid.FromRNode(rn)
	for _, selector := range selectors {
		if !id.IsSelectedBy(selector.ResId) {
			continue
		}
		// TODO: add optimized version for label/annotation selectors:
		// - rn.MatchesAnnotationSelector(selector.AnnotationSelector)
		// - rn.MatchesLabelSelector(selector.LabelSelector)
		// Using these methods directly requires redundant parsing
		// and error handling
		return true
	}
	return false
}
