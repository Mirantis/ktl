package kquery

import (
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Node struct {
	parent     *Node
	rnode      *yaml.RNode
	lazySchema *openapi.ResourceSchema
}

type Nodes struct {
	parent *Nodes
	values []*Node
	lookup *yaml.PathGetter
}

func MakeNodes(rnodes ...*yaml.RNode) *Nodes {
	values := []*Node{}

	for _, rnode := range rnodes {
		values = append(values, &Node{rnode: rnode})
	}

	return &Nodes{values: values}
}

func (nodes *Nodes) Attr(name string) *Nodes {
	lookup := &yaml.PathGetter{Path: []string{name}}
	result := &Nodes{parent: nodes, lookup: lookup}

	for _, node := range nodes.values {
		rnode, err := node.rnode.Pipe(lookup)
		if err != nil {
			continue
		}

		child := &Node{
			parent: node,
			rnode:  rnode,
		}

		result.values = append(result.values, child)
	}

	return result
}
