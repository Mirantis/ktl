package types

import (
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type NodeMeta struct {
	Depth  int
	Node   *yaml.Node
	Parent *NodeMeta
	Schema *openapi.ResourceSchema
	Index  int

	path      NodePath
	mergeKeys []string
}

func (nm *NodeMeta) MergeKeys() []string {
	if nm.mergeKeys == nil && nm.Schema != nil {
		_, nm.mergeKeys = nm.Schema.PatchStrategyAndKeyList()
	}
	return nm.mergeKeys
}

func (nm *NodeMeta) Path() NodePath {
	parent := nm.Parent
	if parent == nil || len(nm.path) > 0 {
		return nm.path
	}
	parentPath := parent.Path()

	switch parent.Node.Kind {
	case yaml.MappingNode:
		nm.path = append(slices.Clone(parentPath), nm.Node.Value)
	case yaml.ScalarNode:
		nm.path = parentPath
	case yaml.SequenceNode:
		keys := parent.MergeKeys()
		if len(keys) < 1 {
			nm.path = append(slices.Clone(parentPath), strconv.FormatInt(int64(nm.Index), 10))
			break
		}

		values := map[string]string{}
		for _, key := range keys {
			values[key] = ""
		}

		for i := 0; i < len(nm.Node.Content); i += 2 {
			key, value := nm.Node.Content[i], nm.Node.Content[i+1]
			if _, isKey := values[key.Value]; !isKey {
				continue
			}
			if value != nil {
				values[key.Value] = value.Value
			}
		}

		parts := []string{}
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%v=%v", key, values[key]))
		}
		part := fmt.Sprintf("[%s]", strings.Join(parts, ","))
		nm.path = append(slices.Clone(parentPath), part)
	}
	return nm.path
}

func AllNodes(yn *yaml.Node) iter.Seq2[*yaml.Node, *NodeMeta] {
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
			if node == nil {
				continue
			}

			switch node.Kind {
			case yaml.MappingNode:
				if len(node.Content)%2 != 0 {
					panic(fmt.Errorf("invalid yaml node: %+v", node))
				}
				for i := len(node.Content) - 2; i >= 0; i -= 2 {
					key, value := node.Content[i], node.Content[i+1]
					var fieldSchema *openapi.ResourceSchema
					if meta.Schema != nil {
						fieldSchema = meta.Schema.Field(key.Value)
					}
					keyMeta := &NodeMeta{
						Node:   key,
						Depth:  meta.Depth + 1,
						Parent: meta,
						Index:  i / 2,
						Schema: fieldSchema,
					}
					valueMeta := &NodeMeta{
						Node:   value,
						Depth:  keyMeta.Depth + 1,
						Parent: keyMeta,
						Index:  i / 2,
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
