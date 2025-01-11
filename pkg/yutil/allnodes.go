package yutil

import (
	"fmt"
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
		nodeStack := []*yaml.Node{yn}
		metaStack := []*NodeMeta{{Depth: 0}}
		for len(nodeStack) > 0 {
			node := nodeStack[len(nodeStack)-1]
			nodeStack = nodeStack[:len(nodeStack)-1]
			meta := metaStack[len(metaStack)-1]
			metaStack = metaStack[:len(metaStack)-1]
			if !yield(node, meta) {
				return
			}
			if node == nil {
				continue
			}

			switch node.Kind {
			case yaml.MappingNode, yaml.DocumentNode:
				if len(node.Content)%2 != 0 {
					panic(fmt.Errorf("invalid yaml node: %+v", node))
				}
				for i := len(node.Content) - 2; i >= 0; i -= 2 {
					key, value := node.Content[i], node.Content[i+1]
					keyMeta := &NodeMeta{
						Depth:      meta.Depth + 1,
						Parent:     node,
						ParentMeta: meta,
					}
					valueMeta := &NodeMeta{
						Depth:      keyMeta.Depth + 1,
						Parent:     key,
						ParentMeta: keyMeta,
					}
					nodeStack = append(nodeStack, value, key)
					metaStack = append(metaStack, valueMeta, keyMeta)
				}
			default:
				for i := len(node.Content) - 1; i >= 0; i-- {
					child := node.Content[i]
					childMeta := &NodeMeta{
						Depth:      meta.Depth + 1,
						Parent:     node,
						ParentMeta: meta,
					}
					nodeStack = append(nodeStack, child)
					metaStack = append(metaStack, childMeta)
				}
			}
		}
	}
}
