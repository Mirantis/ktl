package kstar

import (
	"errors"

	"go.starlark.net/starlark"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	errNotImplemented = errors.New("not implemented")
)

func FromYNode(ynode *yaml.Node) starlark.Value {
	switch kind := ynode.Kind; kind {
	case yaml.MappingNode:
		return &MappingNode{value: ynode}
	case yaml.ScalarNode:
		return &ScalarNode{value: ynode}
	default:
		panic(errNotImplemented)
	}
}
