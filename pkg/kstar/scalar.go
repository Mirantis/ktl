package kstar

import (
	"errors"
	"fmt"
	"strconv"

	"go.starlark.net/starlark"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const ScalarNodeType = "ScalarNode"

var (
	errNotAScalarNode     = errors.New("not a scalar node")
	errInvalidScalarValue = errors.New("invalid scalar value")
)

type ScalarNode struct {
	value  *yaml.Node
	cached starlark.Value
}

var _ starlark.Value = new(ScalarNode)

func (node *ScalarNode) String() string {
	panic(errNotImplemented)
}

func (node *ScalarNode) Type() string {
	return ScalarNodeType
}

func (node *ScalarNode) Freeze() {
	//TODO: freeze
}

func (node *ScalarNode) Truth() starlark.Bool {
	panic(errNotImplemented)
}

func (node *ScalarNode) Hash() (uint32, error) {
	panic(errNotImplemented)
}

func (node *ScalarNode) Value() (starlark.Value, error) {
	var err error
	if node.cached == nil {
		node.cached, err = node.compute()
	}

	return node.cached, err
}

func (node *ScalarNode) compute() (starlark.Value, error) {
	switch tag := node.value.ShortTag(); tag {
	case yaml.NodeTagString:
		return starlark.String(node.value.Value), nil
	case yaml.NodeTagFloat:
		value, err := strconv.ParseFloat(node.value.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errInvalidScalarValue, err)
		}

		return starlark.Float(value), nil
	case yaml.NodeTagInt:
		value, err := strconv.ParseInt(node.value.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errInvalidScalarValue, err)
		}

		return starlark.MakeInt64(value), nil
	case yaml.NodeTagBool:
		value, err := strconv.ParseBool(node.value.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errInvalidScalarValue, err)
		}

		return starlark.Bool(value), nil
	default:
		panic(fmt.Errorf("%w: %s", errNotAScalarNode, tag))
	}
}
