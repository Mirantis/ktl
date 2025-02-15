package filters

import (
	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/resid"
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

func (rule *SkipFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	output := []*yaml.RNode{}
	for _, node := range input {
		res, err := rule.filter(node)
		if err != nil {
			return nil, err
		}
		if res.IsNilOrEmpty() {
			continue
		}
		output = append(output, res)
	}
	return output, nil
}

func (rule *SkipFilter) filter(rn *yaml.RNode) (*yaml.RNode, error) {
	if !rule.match(rn) {
		return rn, nil
	}

	if len(rule.Fields) < 1 {
		return nil, nil
	}

	for _, path := range rule.Fields {
		functions := []yaml.Filter{}
		if len(path) < 1 {
			continue
		}
		name := path[len(path)-1]
		prefix := path[:len(path)-1]
		if len(prefix) > 0 {
			functions = append(functions, yaml.Lookup(prefix...))
		}
		functions = append(functions, yaml.Clear(name))
		rn.Pipe(functions...)
	}

	return rn, nil
}

func (rule *SkipFilter) match(rn *yaml.RNode) bool {
	if len(rule.Resources) > 0 && !matchSelectors(rn, rule.Resources) {
		return false
	}
	if len(rule.Except) > 0 && matchSelectors(rn, rule.Except) {
		return false
	}
	return true
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
