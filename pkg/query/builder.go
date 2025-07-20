package query

import (
	"errors"

	"go.starlark.net/starlark"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	_ starlark.HasAttrs = new(Builder)
)

type Builder struct {
	node *yaml.Node
	err  error
	expr expr
}

func (bld *Builder) Node() (*yaml.Node, error) {
	if bld.node != nil {
		return bld.node, nil
	}

	if bld.err != nil {
		return nil, bld.err
	}

	bld.node, bld.err = bld.expr.eval()

	if bld.node == nil && bld.err == nil {
		bld.node = yaml.MakeNullNode().YNode()
	}

	return bld.node, bld.err
}

func (bld *Builder) Type() string {
	return "Builder"
}

func (bld *Builder) String() string {
	node, _ := bld.Node()
	body, _ := yaml.NewRNode(node).MarshalJSON()
	return string(body)
}

func (bld*Builder) Truth() starlark.Bool {
	node, err := bld.Node()
	if err != nil {
		return false
	}

	return starlark.Bool(!yaml.IsYNodeNilOrEmpty(node))
}

func (bld *Builder) Hash() (uint32, error) {
	//TODO: implement hash
	return 0, errors.New("Builder.Hash() not implemented")
}

func (bld *Builder) Freeze() {
	//TODO: add immutability support
}

func (bld *Builder) AttrNames() []string {
	//TODO: get from Node()
	return nil
}

func (bld *Builder) Attr(name string) (starlark.Value, error) {
	result := &Builder{
		expr: &exprLookup{
			parent: bld,
			key:    name,
		},
	}

	return result, nil
}
