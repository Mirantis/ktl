package dedup

import (
	"fmt"
	"iter"
	"sort"

	"github.com/Mirantis/rekustomize/pkg/yutil"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

func Flatten(rn *yaml.RNode) iter.Seq2[yutil.NodePath, *yaml.RNode] {
	visitor := &flatten{
		paths: map[*yaml.Node]yutil.NodePath{},
	}
	walker := walk.Walker{
		InferAssociativeLists: false, // REVISIT: make configurable
		Sources:               walk.Sources{rn},
		Visitor:               visitor,
	}

	if _, err := walker.Walk(); err != nil {
		panic(err)
	}
	sort.Slice(visitor.entries, func(i, j int) bool {
		return visitor.entries[i].YNode().Line < visitor.entries[j].YNode().Line
	})

	return func(yield func(yutil.NodePath, *yaml.RNode) bool) {
		for _, rn := range visitor.entries {
			if !yield(rn.FieldPath(), rn) {
				return
			}
		}
	}
}

var _ walk.Visitor = (*flatten)(nil)

type flatten struct {
	paths   map[*yaml.Node]yutil.NodePath
	entries []*yaml.RNode
}

func (v *flatten) path(rn *yaml.RNode) yutil.NodePath {
	yn := rn.YNode()
	if yn == nil {
		panic(fmt.Errorf("Node is nil"))
	}
	path := v.paths[yn]
	return path
}

func (v *flatten) setPath(node *yaml.Node, path yutil.NodePath) {
	if node == nil {
		panic(fmt.Errorf("Node is nil"))
	}
	v.paths[node] = path
}

func (v *flatten) add(path yutil.NodePath, node *yaml.RNode) {
	rn := node.Copy()
	rn.AppendToFieldPath(path...)
	v.entries = append(v.entries, rn)
}

func (v *flatten) VisitMap(nodes walk.Sources, rs *openapi.ResourceSchema) (*yaml.RNode, error) {
	rn := nodes.Dest()
	content := rn.Content()
	path := v.path(rn)
	if len(content) == 0 {
		v.add(path, rn)
		return rn, nil
	}
	for i := 0; i < len(content); i += 2 {
		key, node := content[i].Value, content[i+1]
		v.setPath(node, append(append([]string{}, path...), key))
	}
	return rn, nil
}

func (v *flatten) VisitScalar(nodes walk.Sources, rs *openapi.ResourceSchema) (*yaml.RNode, error) {
	rn := nodes.Dest()
	path := v.path(rn)
	v.add(path, rn)
	return rn, nil
}

func (v *flatten) associativeKey(nodes walk.Sources, rs *openapi.ResourceSchema, lk walk.ListKind) string {
	rn := nodes.Dest()
	if lk != walk.AssociativeList || rs == nil {
		return ""
	}
	if _, key := rs.PatchStrategyAndKey(); key != "" {
		return key
	}
	return rn.GetAssociativeKey()
}

func (v *flatten) VisitList(nodes walk.Sources, rs *openapi.ResourceSchema, lk walk.ListKind) (*yaml.RNode, error) {
	rn := nodes.Dest()
	path := v.path(rn)
	if key := v.associativeKey(nodes, rs, lk); key != "" {
		elements, err := rn.Elements()
		if err != nil {
			panic(err)
		}
		for _, node := range elements {
			keyNode := node.Field(key).Value
			keyValue := keyNode.YNode().Value
			epath := append(append([]string{}, path...), fmt.Sprintf("[%s=%s]", key, keyValue))
			v.add(append(epath, key), keyNode)
			v.setPath(node.YNode(), epath)
		}
		return rn, nil
	}
	v.add(path, rn)
	return nil, nil
}
