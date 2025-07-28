package kstar

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Mirantis/ktl/pkg/kquery"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	errNotImplemented = errors.New("not implemented")

	//REVISIT: mimic starlark error
	errIndexOutOfRange = errors.New("index out of range")
)

func FromYNode(ynode *yaml.Node) starlark.Value {
	switch tag := ynode.ShortTag(); tag {
	case yaml.NodeTagMap:
		return &MappingNode{value: ynode}
	case yaml.NodeTagString, yaml.NodeTagInt, yaml.NodeTagFloat, yaml.NodeTagBool:
		return &ScalarNode{value: ynode}
	default:
		panic(errNotImplemented)
	}
}

type Nodes struct {
	query kquery.Nodes
}

var (
	_ starlark.Callable       = new(Nodes)
	_ starlark.HasAttrs       = new(Nodes)
	_ starlark.HasBinary      = new(Nodes)
	_ starlark.HasSetField    = new(Nodes)
	_ starlark.HasSetIndex    = new(Nodes)
	_ starlark.HasSetKey      = new(Nodes)
	_ starlark.TotallyOrdered = new(Nodes)
	_ starlark.Value          = new(Nodes)
)

func (nodes *Nodes) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return nil, errNotImplemented
}

func (nodes *Nodes) Name() string {
	panic(errNotImplemented)
}

func (nodes *Nodes) tryScalar() (starlark.Value, error) {
	if len(nodes.query) != 1 {
		return nodes, nil
	}

	node := nodes.query[0]
	ynode := node.YNode()

	if ynode == nil || ynode.Kind != yaml.ScalarNode {
		return nodes, nil
	}

	switch tag := ynode.ShortTag(); tag {
	case yaml.NodeTagString:
		return starlark.String(ynode.Value), nil
	case yaml.NodeTagInt:
		value, err := strconv.Atoi(ynode.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s node: %w", tag, err)
		}
		return starlark.MakeInt(value), nil
	case yaml.NodeTagFloat:
		value, err := strconv.ParseFloat(ynode.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid %s node: %w", tag, err)
		}
		return starlark.Float(value), nil
	case yaml.NodeTagBool:
		value, err := strconv.ParseBool(ynode.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s node: %w", tag, err)
		}
		return starlark.Bool(value), nil
	default:
		return nil, fmt.Errorf("unsupported node tag: %s", tag)
	}
}

func (nodes *Nodes) Attr(name string) (starlark.Value, error) {
	values := nodes.query.Attr(name)
	result := &Nodes{values}

	return result.tryScalar()
}

func (nodes *Nodes) AttrNames() []string {
	panic(errNotImplemented)
}

func (nodes *Nodes) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
	return nil, errNotImplemented
}

func (nodes *Nodes) SetField(name string, val starlark.Value) error {
	return errNotImplemented
}

func (nodes *Nodes) Index(idx int) starlark.Value {
	if idx < 0 {
		idx = len(nodes.query) - idx
	}

	if idx < 0 || idx >= len(nodes.query) {
		//REVISIT: error in some cases?
		return &Nodes{}
	}

	return &Nodes{kquery.Nodes{nodes.query[idx]}}
}

func (nodes *Nodes) Len() int {
	panic(errNotImplemented)
}

func (nodes *Nodes) SetIndex(idx int, val starlark.Value) error {
	return errNotImplemented
}

func (nodes *Nodes) Get(key starlark.Value) (starlark.Value, bool, error) {
	switch idx := key.(type) {
	case starlark.Int:
		v, ok := idx.Int64()
		if !ok {
			return nil, false, errIndexOutOfRange
		}
		return nodes.Index(int(v)), true, nil
	}
	return nil, false, errNotImplemented
}

func (nodes *Nodes) SetKey(key, val starlark.Value) error {
	return errNotImplemented
}

func (nodes *Nodes) Cmp(y starlark.Value, depth int) (int, error) {
	return 0, errNotImplemented
}

func (nodes *Nodes) String() string {
	panic(errNotImplemented)
}

func (nodes *Nodes) Type() string {
	panic(errNotImplemented)
}

func (nodes *Nodes) Freeze() {
	//FIXME: implement read-only
}

func (nodes *Nodes) Truth() starlark.Bool {
	panic(errNotImplemented)
}

func (nodes *Nodes) Hash() (uint32, error) {
	return 0, errNotImplemented
}
