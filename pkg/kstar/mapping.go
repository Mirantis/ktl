package kstar

import (
	"errors"
	"maps"
	"slices"

	"go.starlark.net/starlark"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const MappingNodeType = "MappingNode"

type MappingNode struct {
	schema *openapi.ResourceSchema
	value  *yaml.Node
	fields map[string]starlark.Value
}

var (
	_ starlark.Value    = new(MappingNode)
	_ starlark.HasAttrs = new(MappingNode)

	errUnsupportedFieldType = errors.New("unsupported field type")
)

func (node *MappingNode) String() string {
	//REVISIT: maybe return json
	panic(errNotImplemented)
}

func (node *MappingNode) Type() string {
	return MappingNodeType
}

func (node *MappingNode) Freeze() {
	//TODO: freeze node
}

func (node *MappingNode) Truth() starlark.Bool {
	return !starlark.Bool(yaml.IsYNodeNilOrEmpty(node.value))
}

func (node *MappingNode) Hash() (uint32, error) {
	panic(errNotImplemented)
}

func (node *MappingNode) loadFields() {
	if node.value == nil {
		node.fields = make(map[string]starlark.Value)
		return
	}

	content := node.value.Content
	node.fields = make(map[string]starlark.Value, len(content)/2)

	for idx := range len(content) / 2 {
		key, value := content[idx*2], content[idx*2+1]
		node.fields[key.Value] = FromYNode(value)
	}
}

func (node *MappingNode) field(name string) (starlark.Value, error) {
	if node.fields == nil {
		node.loadFields()
	}

	field, found := node.fields[name]
	if !found {
		//FIXME: return unsetNode
		return starlark.None, nil
	}

	if scalar, ok := field.(*ScalarNode); ok {
		return scalar.Value()
	}

	return field, nil
}

func (node *MappingNode) Attr(name string) (starlark.Value, error) {
	return node.field(name)
}

func (node *MappingNode) AttrNames() []string {
	if node.fields == nil {
		node.loadFields()
	}

	return slices.Sorted(maps.Keys(node.fields))
}
