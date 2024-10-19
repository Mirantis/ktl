package yutil

import (
	"fmt"
	"iter"
	"slices"
	"sort"

	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

func Flatten(rn *yaml.RNode) iter.Seq2[Path, *yaml.RNode] {
	// FIXME: incorrect parsing (e.g. containerPort)
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
	sort.Sort(visitor)

	return visitor.entries()
}

var _ walk.Visitor = (*flatten)(nil)

type rPathPart struct {
	node           *yaml.RNode
	schema         *openapi.ResourceSchema
	associativeKey string
}

type rPath []rPathPart

func (rpath rPath) path() Path {
	path := Path{}
	for i := 0; i < len(rpath); i++ {
		rpp := rpath[i]
		yn := rpp.node.YNode()
		switch yn.Kind {
		case yaml.ScalarNode:
			path = append(path, yn.Value)
		case yaml.SequenceNode:
			key := rpp.associativeKey
			mapNode := rpath[i+1].node
			keyValue := mapNode.Field(key).Value.YNode().Value
			path = append(path, fmt.Sprintf("[%v=%v]", key, keyValue))
			i++
		default:
			continue
		}
	}
	return path
}

type flatten struct {
	rpath  rPath
	rpaths []rPath
	values []*yaml.RNode
}

func (v *flatten) Len() int {
	return len(v.values)
}

func (v *flatten) Less(i, j int) bool {
	return v.values[i].YNode().Line < v.values[j].YNode().Line
}

func (v *flatten) Swap(i, j int) {
	v.values[i], v.values[j] = v.values[j], v.values[i]
	v.rpaths[i], v.rpaths[j] = v.rpaths[j], v.rpaths[i]
}

func (v *flatten) entries() iter.Seq2[Path, *yaml.RNode] {
	return func(yield func(Path, *yaml.RNode) bool) {
		for i := 0; i < len(v.values); i++ {
			path := v.rpaths[i].path()
			if !yield(path, v.values[i]) {
				return
			}
		}
	}
}

func (v *flatten) trim(column int, kinds ...yaml.Kind) {
	for i := len(v.rpath) - 1; i >= 0; i-- {
		end := v.rpath[i].node.YNode()
		if end.Column < column {
			break
		}
		if end.Column == column && !slices.Contains(kinds, end.Kind) {
			break
		}
		v.rpath = v.rpath[:i]
	}
}

func (v *flatten) pathKind() yaml.Kind {
	if len(v.rpath) < 1 {
		return 0
	}
	rn := v.rpath[len(v.rpath)-1].node
	return rn.YNode().Kind
}

func (v *flatten) pathColumn() int {

	if len(v.rpath) < 1 {
		return 0
	}
	rn := v.rpath[len(v.rpath)-1].node
	return rn.YNode().Column
}

func (v *flatten) popRPath() rPath {
	rpath := slices.Clone(v.rpath)
	if len(v.rpath) > 0 {
		v.rpath = v.rpath[:len(v.rpath)-1]
	}
	return rpath
}

func (v *flatten) append(rpath rPath, rn *yaml.RNode) {
	value := rn.Copy()
	value.ShouldKeep = true
	v.rpaths = append(v.rpaths, rpath)
	v.values = append(v.values, value)
}

func (v *flatten) VisitMap(nodes walk.Sources, rs *openapi.ResourceSchema) (*yaml.RNode, error) {
	rn := nodes.Dest()
	v.trim(rn.YNode().Column, yaml.MappingNode, yaml.SequenceNode, yaml.ScalarNode)
	if rn.IsNilOrEmpty() {
		v.append(v.popRPath(), rn)
		return nil, nil
	}
	v.rpath = append(v.rpath, rPathPart{rn, rs, ""})
	return rn, nil
}

func (v *flatten) VisitScalar(nodes walk.Sources, rs *openapi.ResourceSchema) (*yaml.RNode, error) {
	rn := nodes.Dest()
	v.trim(rn.YNode().Column, yaml.SequenceNode, yaml.ScalarNode)
	if v.pathKind() != yaml.ScalarNode {
		v.rpath = append(v.rpath, rPathPart{rn, rs, ""})
		return rn, nil
	}
	v.append(v.popRPath(), rn)
	return rn, nil
}

func (v *flatten) VisitList(nodes walk.Sources, rs *openapi.ResourceSchema, lk walk.ListKind) (*yaml.RNode, error) {
	rn := nodes.Dest()
	v.trim(rn.YNode().Column)
	switch lk {
	case walk.AssociativeList:
		key := ""
		if rs != nil {
			_, key = rs.PatchStrategyAndKey()
		}
		if key == "" {
			key = rn.GetAssociativeKey()
		}
		if key != "" {
			v.rpath = append(v.rpath, rPathPart{rn, rs, key})
			return rn, nil
		}
		fallthrough
	case walk.NonAssociateList:
		v.append(v.popRPath(), rn)
	}
	return nil, nil
}
