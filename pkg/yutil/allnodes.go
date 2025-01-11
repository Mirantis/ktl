package yutil

import (
	"iter"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func AllNodes(yn *yaml.Node) iter.Seq[*yaml.Node] {
	return func(yield func(*yaml.Node) bool) {
		nodes := []*yaml.Node{yn}
		for len(nodes) > 0 {
			node := nodes[len(nodes)-1]
			nodes = nodes[:len(nodes)-1]
			if node == nil {
				continue
			}
			if !yield(node) {
				return
			}
			for i := len(node.Content) - 1; i >= 0; i-- {
				next := node.Content[i]
				nodes = append(nodes, next)
			}
		}
	}
}
