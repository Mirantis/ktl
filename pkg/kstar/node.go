package kstar

import (
	"errors"
	"fmt"
	"strconv"

	"go.starlark.net/starlark"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	errNotImplemented  = errors.New("not implemented")
	errUnsupportedType = errors.New("unsupported type")
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

func fromStarlarkEntries(value starlark.IterableMapping) (*yaml.Node, error) {
	ynode := yaml.NewMapRNode(nil).YNode()
	ynode.Tag = yaml.NodeTagMap

	for itemKey, itemValue := range starlark.Entries(value) {
		keyYNode, err := FromStarlark(itemKey)
		if err != nil {
			return nil, err
		}

		valueYNode, err := FromStarlark(itemValue)
		if err != nil {
			return nil, err
		}

		ynode.Content = append(ynode.Content, keyYNode, valueYNode)
	}

	return ynode, nil
}

func fromStarlarkElements(value starlark.Iterable) (*yaml.Node, error) {
	ynode := yaml.NewListRNode().YNode()
	ynode.Tag = yaml.NodeTagSeq

	for item := range starlark.Elements(value) {
		itemYNode, err := FromStarlark(item)
		if err != nil {
			return nil, err
		}

		ynode.Content = append(ynode.Content, itemYNode)
	}

	return ynode, nil
}

func FromStarlark(value starlark.Value) (*yaml.Node, error) {
	switch value := value.(type) {
	case *ScalarNode:
		return yaml.CopyYNode(value.value), nil
	case *MappingNode:
		return yaml.CopyYNode(value.value), nil
	case starlark.String:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   yaml.NodeTagString,
			Value: value.GoString(),
		}, nil
	case starlark.Float:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   yaml.NodeTagFloat,
			Value: value.String(),
		}, nil
	case starlark.Int:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   yaml.NodeTagInt,
			Value: value.String(),
		}, nil
	case starlark.Bool:
		ynode := yaml.NewScalarRNode(strconv.FormatBool(bool(value))).YNode()
		ynode.Tag = yaml.NodeTagBool
		return ynode, nil
	case starlark.IterableMapping:
		return fromStarlarkEntries(value)
	case starlark.Iterable:
		return fromStarlarkElements(value)
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedType, value.Type())
	}
}
