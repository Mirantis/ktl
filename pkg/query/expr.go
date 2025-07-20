package query

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type expr interface {
	eval() (*yaml.Node, error)
}

type exprLookup struct {
	parent *Builder
	key    string
}

func (lookup *exprLookup) eval() (*yaml.Node, error) {
	node, err := lookup.parent.Node()
	if err != nil {
		return nil, err
	}

	if node == nil {
		return nil, nil
	}

	var filter yaml.Filter
	switch node.Kind {
	case yaml.MappingNode:
		filter = &yaml.PathGetter{
			Path: []string{lookup.key},
		}
	case yaml.SequenceNode:
		filter = &yaml.PathMatcher{
			Path: []string{"*", lookup.key},
		}
	default:
		return nil, nil
	}

	result, err := yaml.NewRNode(node).Pipe(filter)
	if err != nil {
		// not found is not an errror
		return nil, nil
	}

	return result.YNode(), nil
}
