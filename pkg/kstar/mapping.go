package kstar

import (
	"errors"
	"fmt"
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
	_ starlark.Value       = new(MappingNode)
	_ starlark.HasSetField = new(MappingNode)
	_ starlark.HasSetKey   = new(MappingNode)

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

func (node *MappingNode) SetField(name string, value starlark.Value) error {
	if node.fields != nil && node.fields[name] == value {
		return nil
	}

	newYNode, err := FromStarlark(value)
	if err != nil {
		return fmt.Errorf("unable to set %q: %w", name, err)
	}

	if node.fields != nil {
		node.fields[name] = FromYNode(newYNode)
	}

	newRNode := yaml.NewRNode(newYNode)
	keyRNode := yaml.NewStringRNode(name)
	thisRNode := yaml.NewRNode(node.value)

	return thisRNode.PipeE(yaml.MapEntrySetter{
		Name:  name,
		Key:   keyRNode,
		Value: newRNode,
	})
}

func (node *MappingNode) Get(key starlark.Value) (_ starlark.Value, found bool, _ error) {
	switch key := key.(type) {
	case starlark.String:
		value, err := node.Attr(key.GoString())
		return value, true, err
	case *MappingNode:
		//TODO: add match lookup
		panic(errNotImplemented)
	default:
		return nil, false, fmt.Errorf(
			"%w: %q",
			errUnsupportedFieldType,
			key.Type(),
		)
	}
}

func (node *MappingNode) SetKey(key, value starlark.Value) error {
	switch key := key.(type) {
	case starlark.String:
		field := key.GoString()

		return node.SetField(field, value)
	case *MappingNode:
		//TODO: add match lookup
		panic(errNotImplemented)
	default:
		return fmt.Errorf(
			"%w: %q",
			errUnsupportedFieldType,
			key.Type(),
		)
	}
}
