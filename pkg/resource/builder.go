package resource

import (
	"fmt"
	"slices"

	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Builder struct {
	path  types.NodePath
	nodes []*yaml.RNode

	Hit, Miss int
}

func NewNodeBuilder(root *yaml.RNode) *Builder {
	return &Builder{nodes: []*yaml.RNode{root}}
}

func NewBuilder(resID resid.ResId) *Builder {
	resNode := yaml.NewMapRNode(nil)
	resNode.SetApiVersion(resID.ApiVersion())
	resNode.SetKind(resID.Kind)
	resNode.SetName(resID.Name) //nolint:errcheck

	if resID.Namespace != "" {
		resNode.SetNamespace(resID.Namespace) //nolint:errcheck
	}

	return NewNodeBuilder(resNode)
}

func (b *Builder) RNode() *yaml.RNode {
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
		b.Miss++

		return b.nodes[0], path
	}

	if common > 0 {
		b.Hit++
	}

	return b.nodes[common], path[common:]
}

func (b *Builder) Add(path types.NodePath, kind yaml.Kind) (*yaml.RNode, error) {
	root, sub := b.skipCommon(path)

	resNode, err := root.Pipe(yaml.LookupCreate(kind, sub...))
	if err != nil {
		return nil, fmt.Errorf("unable to add resource attribute: %w", err)
	}

	b.nodes[len(path)] = resNode

	return resNode, nil
}

func (b *Builder) Set(path types.NodePath, node *yaml.Node) (*yaml.RNode, error) {
	resNode, err := b.Add(path, node.Kind)
	if err != nil {
		return nil, err
	}

	resNode.SetYNode(node)

	return resNode, nil
}
