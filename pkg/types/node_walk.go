package types

import (
	"fmt"
	"iter"

	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func WalkNode(yn *yaml.Node) iter.Seq2[*yaml.Node, *NodeMeta] {
	id := resid.FromRNode(yaml.NewRNode(yn))
	schema := openapi.SchemaForResourceType(id.AsTypeMeta())
	return func(yield func(*yaml.Node, *NodeMeta) bool) {
		metaStack := []*NodeMeta{{Node: yn, Schema: schema}}
		for len(metaStack) > 0 {
			var node *yaml.Node
			meta := metaStack[len(metaStack)-1]
			metaStack = metaStack[:len(metaStack)-1]
			if meta != nil {
				node = meta.Node
			}
			if !yield(node, meta) {
				return
			}
			if node == nil || meta.IsLeaf() {
				continue
			}

			switch node.Kind {
			case yaml.MappingNode:
				if len(node.Content)%2 != 0 {
					panic(fmt.Errorf("invalid yaml node: %+v", node))
				}
				for i := len(node.Content)/2 - 1; i >= 0; i-- {
					key, value := node.Content[2*i], node.Content[2*i+1]
					var fieldSchema *openapi.ResourceSchema
					if meta.Schema != nil {
						fieldSchema = meta.Schema.Field(key.Value)
					}
					keyMeta := &NodeMeta{
						Node:   key,
						Depth:  meta.Depth + 1,
						Parent: meta,
						Index:  i,
						Schema: fieldSchema,
					}
					valueMeta := &NodeMeta{
						Node:   value,
						Depth:  keyMeta.Depth + 1,
						Parent: keyMeta,
						Index:  i,
						Schema: fieldSchema,
					}
					metaStack = append(metaStack, valueMeta)
				}
			default:
				var elementSchema *openapi.ResourceSchema
				if meta.Schema != nil && len(node.Content) > 0 {
					elementSchema = meta.Schema.Elements()
				}
				for i := len(node.Content) - 1; i >= 0; i-- {
					child := node.Content[i]
					childMeta := &NodeMeta{
						Node:   child,
						Depth:  meta.Depth + 1,
						Parent: meta,
						Index:  i,
						Schema: elementSchema,
					}
					metaStack = append(metaStack, childMeta)
				}
			}
		}
	}
}
