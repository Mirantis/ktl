package yutil

import (
	"iter"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func AllNodes(yn *yaml.Node) iter.Seq2[*yaml.Node, int] {
	return func(yield func(*yaml.Node, int) bool) {
		nodes := []*yaml.Node{yn}
		depths := []int{0}
		for len(nodes) > 0 {
			node := nodes[len(nodes)-1]
			nodes = nodes[:len(nodes)-1]
			depth := depths[len(depths)-1]
			depths = depths[:len(depths)-1]
			if node == nil {
				continue
			}
			if !yield(node, depth) {
				return
			}
			elementSize := 1
			if node.Kind == yaml.MappingNode || node.Kind == yaml.DocumentNode {
				elementSize = 2
			}
			for i := len(node.Content) - 1; i >= 0; i-- {
				next := node.Content[i]
				nodes = append(nodes, next)
				depths = append(depths, depth+(i%elementSize)+1)
			}
		}
	}
}
