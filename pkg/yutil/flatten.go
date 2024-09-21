package yutil

import (
	"fmt"
	"iter"
	"slices"

	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

func Flatten(rn *yaml.RNode) iter.Seq2[Path, *yaml.RNode] {
	visitor := &flatten{}
	walker := walk.Walker{
		InferAssociativeLists: false, // REVISIT: make configurable
		VisitKeysAsScalars:    true,

		Sources: walk.Sources{rn},
		Visitor: visitor,
	}

	if _, err := walker.Walk(); err != nil {
		panic(err)
	}

	return visitor.entries()
}

var _ walk.Visitor = (*flatten)(nil)

type flatten struct {
	rpath  []*yaml.RNode
	rpaths [][]*yaml.RNode
	values []*yaml.RNode
}

func pathFromRPath(rpath []*yaml.RNode) Path {
	path := Path{}
	for i := 0; i < len(rpath); i++ {
		rn := rpath[i]
		yn := rn.YNode()
		switch yn.Kind {
		case yaml.ScalarNode:
			path = append(path, yn.Value)
		case yaml.SequenceNode:
			key := rn.GetAssociativeKey()
			mapNode := rpath[i+1]
			keyValue := mapNode.Field(key).Value.YNode().Value
			path = append(path, fmt.Sprintf("[%v=%v]", key, keyValue))
			i++
		default:
			continue
		}
	}
	return path
}

func (v *flatten) entries() iter.Seq2[Path, *yaml.RNode] {
	return func(yield func(Path, *yaml.RNode) bool) {
		for i := 0; i < len(v.values); i++ {
			path := pathFromRPath(v.rpaths[i])
			if !yield(path, v.values[i]) {
				return
			}
		}
	}
}

func (v *flatten) trim(column int) {
	for i := len(v.rpath) - 1; i >= 0; i-- {
		end := v.rpath[i]
		if end.YNode().Column <= column {
			return
		}
		v.rpath = v.rpath[:i]
	}
}

func (v *flatten) pathKind() yaml.Kind {
	if len(v.rpath) < 1 {
		return 0
	}
	rn := v.rpath[len(v.rpath)-1]
	return rn.YNode().Kind
}

func (v *flatten) pathColumn() int {

	if len(v.rpath) < 1 {
		return 0
	}
	rn := v.rpath[len(v.rpath)-1]
	return rn.YNode().Column
}

func (v *flatten) popRPath() []*yaml.RNode {
	rpath := slices.Clone(v.rpath)
	if len(v.rpath) > 0 {
		v.rpath = v.rpath[:len(v.rpath)-1]
	}
	return rpath
}

func (v *flatten) VisitMap(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	rn := nodes.Dest()
	v.trim(rn.YNode().Column)
	if v.pathKind() == rn.YNode().Kind {
		// associative list entry
		v.rpath[len(v.rpath)-1] = rn
		return rn, nil
	}
	v.rpath = append(v.rpath, rn)
	return rn, nil
}

func (v *flatten) VisitScalar(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	rn := nodes.Dest()
	v.trim(rn.YNode().Column)
	if v.pathKind() != yaml.ScalarNode {
		v.rpath = append(v.rpath, rn)
		return rn, nil
	}
	if v.pathColumn() == rn.YNode().Column {
		v.popRPath()
		v.rpath = append(v.rpath, rn)
		return rn, nil
	}
	v.rpaths = append(v.rpaths, v.popRPath())
	v.values = append(v.values, rn)
	return rn, nil
}

func (v *flatten) VisitList(nodes walk.Sources, _ *openapi.ResourceSchema, lk walk.ListKind) (*yaml.RNode, error) {
	rn := nodes.Dest()
	v.trim(rn.YNode().Column)
	switch lk {
	case walk.NonAssociateList:
		v.rpaths = append(v.rpaths, v.popRPath())
		v.values = append(v.values, rn)
	case walk.AssociativeList:
		v.rpath = append(v.rpath, rn)
	}
	return rn, nil
}
