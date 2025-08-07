package kstar

import (
	"fmt"

	"go.starlark.net/starlark"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const SequenceNodeType = "SequenceNode"

type SequenceNode struct {
	schema *NodeSchema
	ynode  *yaml.Node
	elems  []starlark.Value
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
	return !starlark.Bool(yaml.IsYNodeNilOrEmpty(node.ynode))
}

func (node *SequenceNode) Hash() (uint32, error) {
	panic(errNotImplemented)
}

func (node *SequenceNode) setSchema(ns *NodeSchema) {
	node.schema = ns
}

func (node *SequenceNode) len() int {
	if node.ynode == nil {
		return 0
	}

	return len(node.ynode.Content)
}

func (node *SequenceNode) loadElements() {
	if node.ynode == nil {
		node.elems = []starlark.Value{}
	}

	schema := node.schema.Elements()
	elems := make([]starlark.Value, len(node.ynode.Content))
	for idx, ynode := range node.ynode.Content {
		value := FromYNode(ynode)
		value.setSchema(schema)
		elems[idx] = value
	}

	node.elems = elems
}

func (node *SequenceNode) index(idx int) starlark.Value {
	if node.elems == nil {
		node.loadElements()
	}

	value := node.elems[idx]

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
	if node.elems == nil {
		node.loadElements()
	}

	switch key := key.(type) {
	case starlark.Int:
		idx, ok := node.toIndex(key)
		if !ok {
			return fmt.Errorf("%w: %s", errInvalid, key.String())
		}

		ynode, err := FromStarlark(value)
		if err != nil {
			return err
		}

		schema := node.schema.Elements()
		elem := FromYNode(ynode)
		elem.setSchema(schema)

		node.ynode.Content[idx] = ynode
		node.elems[idx] = elem

		return nil
	default:
		return fmt.Errorf("%w: %s", errUnsupportedType, key.Type())
	}
}

func (node *SequenceNode) Iterate() starlark.Iterator {
	return &seqIterator{node: node}
}

type seqIterator struct {
	node *SequenceNode
	curr int
}

func (iter *seqIterator) Next(value *starlark.Value) bool {
	if iter.curr >= iter.node.len() {
		return false
	}

	*value = iter.node.index(iter.curr)
	iter.curr++

	return true
}

func (iter *seqIterator) Done() {
	iter.node = nil
}

func (node *SequenceNode) filter(th *starlark.Thread, fn starlark.Callable) (*SequenceNode, error) {
	if node.elems == nil {
		node.loadElements()
	}

	ynodes := []*yaml.Node{}
	elems := []starlark.Value{}

	for idx := range node.len() {
		args := starlark.Tuple{node.index(idx)}

		result, err := fn.CallInternal(th, args, nil)
		if err != nil {
			return nil, err
		}

		if !result.Truth() {
			continue
		}

		ynodes = append(ynodes, node.ynode.Content[idx])
		elems = append(elems, node.elems[idx])
	}

	return &SequenceNode{
		schema: node.schema,
		ynode: &yaml.Node{
			Kind:    yaml.SequenceNode,
			Tag:     yaml.NodeTagSeq,
			Content: ynodes,
		},
		elems: elems,
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
