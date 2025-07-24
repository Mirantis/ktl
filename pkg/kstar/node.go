package kstar

import (
	"errors"

	"github.com/Mirantis/ktl/pkg/kquery"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

var errNotImplemented = errors.New("not implemented")

type Nodes struct {
	query *kquery.Nodes
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

func (nodes *Nodes) Attr(name string) (starlark.Value, error) {
	return nil, errNotImplemented
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
	panic(errNotImplemented)
}

func (nodes *Nodes) Len() int {
	panic(errNotImplemented)
}

func (nodes *Nodes) SetIndex(idx int, val starlark.Value) error {
	return errNotImplemented
}

func (nodes *Nodes) Get(key starlark.Value) (starlark.Value, bool, error) {
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
	panic(errNotImplemented)
}

func (nodes *Nodes) Truth() starlark.Bool {
	panic(errNotImplemented)
}

func (nodes *Nodes) Hash() (uint32, error) {
	return 0, errNotImplemented
}
