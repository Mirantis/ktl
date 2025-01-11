package yutil

import (
	"iter"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type NodeMeta struct {
	Depth      int
	Parent     *yaml.Node
	ParentMeta *NodeMeta
}

func AllNodes(yn *yaml.Node) iter.Seq2[*yaml.Node, *NodeMeta] {
	return func(yield func(*yaml.Node, *NodeMeta) bool) {
		nodes := []*yaml.Node{yn}
		metas := []*NodeMeta{{Depth: 0}}
		for len(nodes) > 0 {
			node := nodes[len(nodes)-1]
			nodes = nodes[:len(nodes)-1]
			meta := metas[len(metas)-1]
			metas = metas[:len(metas)-1]
			if node == nil {
				continue
			}
			if !yield(node, meta) {
				return
			}
			isKVNode := node.Kind == yaml.MappingNode || node.Kind == yaml.DocumentNode
			for i := len(node.Content) - 1; i >= 0; i-- {
				next := node.Content[i]
				nextMeta := &NodeMeta{
					Depth:      meta.Depth + 1,
					Parent:     node,
					ParentMeta: meta,
				}
				if isKVNode && i%2 == 0 {
					valueMeta := metas[len(metas)-1]
					valueMeta.Parent = next
					valueMeta.ParentMeta = nextMeta
					valueMeta.Depth = nextMeta.Depth + 1
				}
				nodes = append(nodes, next)
				metas = append(metas, nextMeta)
			}
		}
	}
}
