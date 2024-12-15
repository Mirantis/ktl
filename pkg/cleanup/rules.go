package cleanup

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func DefaultRules() Rules {
	rules := []Rule{&schemaRule{}}
	return rules
}

type Rule interface {
	Apply(*yaml.RNode) error
}

type Rules []Rule

func (r Rules) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	result := []*yaml.RNode{}
	for _, rn := range nodes {
		for _, rule := range r {
			err := rule.Apply(rn)
			if err != nil {
				return nil, err
			}
			if rn.IsNil() {
				break
			}
		}
		if !rn.IsNil() {
			result = append(result, rn)
		}
	}
	return result, nil
}
