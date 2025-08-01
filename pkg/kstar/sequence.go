package kstar

import (
	"fmt"

	"go.starlark.net/starlark"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const SequenceNodeType = "SequenceNode"

type SequenceNode struct {
	schema *openapi.ResourceSchema
	value  *yaml.Node
}

var (
	_ starlark.Value     = new(SequenceNode)
	_ starlark.HasSetKey = new(SequenceNode)
)

func (node *SequenceNode) String() string {
	//REVISIT: maybe return json
	panic(errNotImplemented)
}

func (node *SequenceNode) Type() string {
	return SequenceNodeType
}

func (node *SequenceNode) Freeze() {
	//TODO: freeze node
}

func (node *SequenceNode) Truth() starlark.Bool {
	return !starlark.Bool(yaml.IsYNodeNilOrEmpty(node.value))
}

func (node *SequenceNode) Hash() (uint32, error) {
	panic(errNotImplemented)
}

func (node *SequenceNode) len() int {
	if node.value == nil {
		return 0
	}

	return len(node.value.Content)
}

func (node *SequenceNode) index(idx int) starlark.Value {
	//TODO: add cache
	return FromYNode(node.value.Content[idx])
}

func (node *SequenceNode) toIndex(key starlark.Int) (int, bool) {
	idx64, ok := key.Int64()
	if !ok {
		return -1, false
	}

	idx := int(idx64)
	if idx < 0 {
		idx = node.len() + idx
	}

	if idx < 0 || idx >= node.len() {
		return -1, false
	}

	return idx, true
}

func (node *SequenceNode) Get(key starlark.Value) (_ starlark.Value, found bool, _ error) {
	switch key := key.(type) {
	case starlark.Int:
		idx, ok := node.toIndex(key)
		if !ok {
			return nil, false, nil
		}

		value := node.index(idx)
		if scalar, ok := value.(*ScalarNode); ok {
			value, err := scalar.Value()
			return value, err == nil, err
		}

		return node.index(idx), true, nil
	default:
		return nil, false, fmt.Errorf("%w: %s", errUnsupportedType, key.Type())
	}
}

func (node *SequenceNode) SetKey(key, value starlark.Value) error {
	switch key := key.(type) {
	case starlark.Int:
		idx, ok := node.toIndex(key)
		if !ok {
			return fmt.Errorf("%w: %s", errInvalid, key.String())
		}

		item, err := FromStarlark(value)
		if err != nil {
			return err
		}

		node.value.Content[idx] = item

		return nil
	default:
		return fmt.Errorf("%w: %s", errUnsupportedType, key.Type())
	}
}
