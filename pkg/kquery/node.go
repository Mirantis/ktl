package kquery

import (
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Node struct {
	parent     *Node
	rnode      *yaml.RNode
	lazySchema *openapi.ResourceSchema
	lookup     *yaml.PathGetter
}

func (node *Node) YNode() *yaml.Node {
	if node == nil {
		return nil
	}

	return node.rnode.YNode()
}

func (node *Node) ensureRNode(kind yaml.Kind) *yaml.RNode {
	if node == nil {
		return nil
	}

	if node.rnode != nil {
		return node.rnode
	}

	if node.lookup == nil {
		return nil
	}

	if kind != 0 {
		node.lookup.Create = kind
	}

	node.rnode, _ = node.parent.ensureRNode(0).Pipe(node.lookup)

	return node.rnode
}

func (node *Node) setCreate(kind yaml.Kind) {
	if node.rnode != nil {
		return
	}

	if node.lookup == nil {
		return
	}

	if node.lookup.Create != 0 {
		//REVISIT: maybe error if != kind
		return
	}

	node.lookup.Create = kind
}

type Nodes []*Node

func MakeNodes(rnodes ...*yaml.RNode) Nodes {
	nodes := Nodes{}

	for _, rnode := range rnodes {
		nodes = append(nodes, &Node{rnode: rnode})
	}

	return nodes
}

func (nodes Nodes) Attr(name string) Nodes {
	result := Nodes{}
	lookup := &yaml.PathGetter{Path: []string{name}}

	for _, node := range nodes {
		node.setCreate(yaml.MappingNode)

		//TODO: cache and/or unique - for Pipe() and child
		rnode, err := node.rnode.Pipe(lookup)
		if err != nil {
			rnode = nil
		}

		child := &Node{
			parent: node,
			rnode:  rnode,
			lookup: lookup,
		}

		result = append(result, child)
	}

	return result
}

func (nodes Nodes) SetValue(value *yaml.Node) {
	var kind yaml.Kind
	if value != nil {
		kind = value.Kind
	}

	for _, node := range nodes {
		rnode := node.ensureRNode(kind)
		if rnode == nil {
			continue
		}

		rnode.SetYNode(yaml.CopyYNode(value))
	}
}
