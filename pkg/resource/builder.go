package resource

import (
	"slices"

	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Builder struct {
	path  types.NodePath
	nodes []*yaml.RNode

	hit, miss int
}

func NewNodeBuilder(root *yaml.RNode) *Builder {
	return &Builder{nodes: []*yaml.RNode{root}}
}

func NewBuilder(id resid.ResId) *Builder {
	rn := yaml.NewMapRNode(nil)
	rn.SetApiVersion(id.ApiVersion())
	rn.SetKind(id.Kind)
	rn.SetName(id.Name)
	if id.Namespace != "" {
		rn.SetNamespace(id.Namespace)
	}
	return NewNodeBuilder(rn)
}

func (b *Builder) Build() *yaml.RNode {
	return b.nodes[0]
}

func (b *Builder) skipCommon(path types.NodePath) (*yaml.RNode, types.NodePath) {
	common := 0
	for i := range min(len(path), len(b.path)) {
		if path[i] != b.path[i] {
			break
		}
		common++
	}

	b.path = slices.Clone(path)
	padding := slices.Repeat([]*yaml.RNode{nil}, len(path[common:]))
	b.nodes = append(b.nodes[:1+common], padding...)

	if b.nodes[common] == nil {
		b.miss++
		return b.nodes[0], path
	}
	if common > 0 {
		b.hit++
	}
	return b.nodes[common], path[common:]
}

func (b *Builder) Add(path types.NodePath, kind yaml.Kind) (*yaml.RNode, error) {
	root, sub := b.skipCommon(path)
	rn, err := root.Pipe(yaml.LookupCreate(kind, sub...))
	if err != nil {
		return nil, err
	}
	b.nodes[len(path)] = rn
	return rn, nil
}

func (b *Builder) Set(path types.NodePath, node *yaml.Node) (*yaml.RNode, error) {
	rn, err := b.Add(path, node.Kind)
	if err != nil {
		return nil, err
	}
	rn.SetYNode(node)
	return rn, nil
}
