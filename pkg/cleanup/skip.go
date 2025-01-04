package cleanup

import (
	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyutil "sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func NewSkipRule(resources, excluding []*types.Selector, fields []string) (Rule, error) {
	paths := []types.NodePath{}
	for _, field := range fields {
		path := types.NodePath(kyutil.SmarterPathSplitter(field, "."))
		paths = append(paths, path)
	}
	rule := &skipRule{
		resources: resources,
		excluding: excluding,
		paths:     paths,
	}
	return rule, nil
}

type skipRule struct {
	resources []*types.Selector
	excluding []*types.Selector
	paths     []types.NodePath
}

func (rule *skipRule) Apply(rn *yaml.RNode) error {
	if !rule.match(rn) {
		return nil
	}

	if len(rule.paths) < 1 {
		rn.SetYNode(nil)
		return nil
	}

	for _, path := range rule.paths {
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

func (rule *skipRule) match(rn *yaml.RNode) bool {
	if len(rule.resources) > 0 && !matchSelectors(rn, rule.resources) {
		return false
	}
	if len(rule.excluding) > 0 && matchSelectors(rn, rule.excluding) {
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
