package kstar

import (
	"errors"

	"github.com/Mirantis/ktl/pkg/kquery"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

var errNotImplemented = errors.New("not implemented")

type NodeSet struct {
	nodes *kquery.NodeSet
}

var (
	_ starlark.Callable       = new(NodeSet)
	_ starlark.HasAttrs       = new(NodeSet)
	_ starlark.HasBinary      = new(NodeSet)
	_ starlark.HasSetField    = new(NodeSet)
	_ starlark.HasSetIndex    = new(NodeSet)
	_ starlark.HasSetKey      = new(NodeSet)
	_ starlark.TotallyOrdered = new(NodeSet)
	_ starlark.Value          = new(NodeSet)
)

func (nset *NodeSet) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	return nil, errNotImplemented
}

func (nset *NodeSet) Name() string {
	panic(errNotImplemented)
}

func (nset *NodeSet) Attr(name string) (starlark.Value, error) {
	return nil, errNotImplemented
}

func (nset *NodeSet) AttrNames() []string {
	panic(errNotImplemented)
}

func (nset *NodeSet) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
	return nil, errNotImplemented
}

func (nset *NodeSet) SetField(name string, val starlark.Value) error {
	return errNotImplemented
}

func (nset *NodeSet) Index(idx int) starlark.Value {
	panic(errNotImplemented)
}

func (nset *NodeSet) Len() int {
	panic(errNotImplemented)
}

func (nset *NodeSet) SetIndex(idx int, val starlark.Value) error {
	return errNotImplemented
}

func (nset *NodeSet) Get(key starlark.Value) (starlark.Value, bool, error) {
	return nil, false, errNotImplemented
}

func (nset *NodeSet) SetKey(key, val starlark.Value) error {
	return errNotImplemented
}

func (nset *NodeSet) Cmp(y starlark.Value, depth int) (int, error) {
	return 0, errNotImplemented
}

func (nset *NodeSet) String() string {
	panic(errNotImplemented)
}

func (nset *NodeSet) Type() string {
	panic(errNotImplemented)
}

func (nset *NodeSet) Freeze() {
	panic(errNotImplemented)
}

func (nset *NodeSet) Truth() starlark.Bool {
	panic(errNotImplemented)
}

func (nset *NodeSet) Hash() (uint32, error) {
	return 0, errNotImplemented
}
