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
	_ starlark.Iterable  = new(SequenceNode)
	_ starlark.Callable  = new(SequenceNode)
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
	value := FromYNode(node.value.Content[idx])

	scalar, isScalar := value.(*ScalarNode)
	if !isScalar {
		return value
	}

	scalarValue, err := scalar.Value()
	if err != nil {
		return value
	}

	return scalarValue
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

func (node *SequenceNode) Iterate() starlark.Iterator {
	iter := &seqIterator{}

	if node == nil {
		return iter
	}

	if node.value != nil {
		iter.ynodes = append(iter.ynodes, node.value.Content...)
	}

	if node.schema != nil {
		iter.schema = node.schema.Elements()
	}

	return iter
}

type seqIterator struct {
	schema *openapi.ResourceSchema
	ynodes []*yaml.Node
}

func (iter *seqIterator) Next(value *starlark.Value) bool {
	if iter == nil || len(iter.ynodes) < 1 {
		return false
	}

	*value = FromYNode(iter.ynodes[0])
	iter.ynodes = iter.ynodes[1:]

	scalar, isScalar := (*value).(*ScalarNode)
	if !isScalar {
		return true
	}

	scalarValue, err := scalar.Value()
	if err != nil {
		return true
	}

	*value = scalarValue

	return true
}

func (iter *seqIterator) Done() {
	iter.ynodes = nil
}

func (node *SequenceNode) filter(th *starlark.Thread, fn starlark.Callable) (*SequenceNode, error) {
	items := []*yaml.Node{}

	for idx := range node.len() {
		ynode := node.value.Content[idx]
		value := node.index(idx)

		result, err := fn.CallInternal(th, starlark.Tuple{value}, nil)
		if err != nil {
			return nil, err
		}

		if !result.Truth() {
			continue
		}

		items = append(items, ynode)
	}

	return &SequenceNode{
		schema: node.schema,
		value: &yaml.Node{
			Kind:    yaml.SequenceNode,
			Tag:     yaml.NodeTagSeq,
			Content: items,
		},
	}, nil
}

func (node *SequenceNode) Name() string { // Callable Name
	return "Filter" + SequenceNodeType
}

func (node *SequenceNode) CallInternal(th *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var fn starlark.Callable

	err := starlark.UnpackPositionalArgs(node.Name(), args, kwargs, 1, &fn)
	if err != nil {
		return nil, err
	}

	return node.filter(th, fn)
}
