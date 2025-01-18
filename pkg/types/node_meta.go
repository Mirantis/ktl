package types

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/openapi"
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

func (nm *NodeMeta) IsLeaf() bool {
	switch nm.Node.Kind {
	case yaml.SequenceNode:
		isNonAssociativeList := (len(nm.MergeKeys()) == 0)
		return isNonAssociativeList
	case yaml.ScalarNode:
		parentIsScalar := (nm.Parent != nil && nm.Parent.Node.Kind == yaml.ScalarNode)
		return parentIsScalar
	default:
		return false
	}
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
