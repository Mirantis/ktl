package cleanup

import (
	"fmt"

	"github.com/Mirantis/rekustomize/pkg/yutil"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyutil "sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func NewSkipRule(resources, excluding []*types.Selector, fields []string) (Rule, error) {
	rsr, err := newSelectorsRegex(resources)
	if err != nil {
		return nil, err
	}
	esr, err := newSelectorsRegex(excluding)
	if err != nil {
		return nil, err
	}
	paths := []yutil.NodePath{}
	for _, field := range fields {
		path := yutil.NodePath(kyutil.SmarterPathSplitter(field, "."))
		paths = append(paths, path)
	}
	rule := &skipRule{
		resources: rsr,
		excluding: esr,
		paths:     paths,
	}
	return rule, nil
}

type skipRule struct {
	resources selectorsRegex
	excluding selectorsRegex
	paths     []yutil.NodePath
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
	if len(rule.resources) > 0 && !rule.resources.match(rn) {
		return false
	}
	if len(rule.excluding) > 0 && rule.excluding.match(rn) {
		return false
	}
	return true
}

type selectorsRegex []*types.SelectorRegex

func newSelectorsRegex(selectors []*types.Selector) (selectorsRegex, error) {
	ret := selectorsRegex{}
	for _, selector := range selectors {
		sr, err := types.NewSelectorRegex(selector)
		if err != nil {
			return nil, fmt.Errorf("invalid selector %+v: %v", selector, err)
		}
		ret = append(ret, sr)
	}
	return ret, nil
}

func (sr selectorsRegex) match(rn *yaml.RNode) bool {
	for _, selector := range sr {
		id := resid.FromRNode(rn)
		if !selector.MatchGvk(id.Gvk) {
			continue
		}
		if !selector.MatchNamespace(id.Namespace) {
			continue
		}
		if !selector.MatchName(id.Name) {
			continue
		}
		return true
	}
	return false
}
