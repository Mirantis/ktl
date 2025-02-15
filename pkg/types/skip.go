package types

import (
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type SkipRule struct {
	Resources []*Selector `json:"resources" yaml:"resources"`
	Except    []*Selector `json:"except" yaml:"except"`
	Fields    []NodePath  `json:"fields" yaml:"fields"`
}

func (rule *SkipRule) Apply(rn *yaml.RNode) error {
	if !rule.match(rn) {
		return nil
	}

	if len(rule.Fields) < 1 {
		rn.SetYNode(nil)
		return nil
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

	return nil
}

func (rule *SkipRule) match(rn *yaml.RNode) bool {
	if len(rule.Resources) > 0 && !matchSelectors(rn, rule.Resources) {
		return false
	}
	if len(rule.Except) > 0 && matchSelectors(rn, rule.Except) {
		return false
	}
	return true
}

func matchSelectors(rn *yaml.RNode, selectors []*Selector) bool {
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
