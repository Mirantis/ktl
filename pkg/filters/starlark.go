package filters

import (
	"log/slog"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/types"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type StarlarkFilter struct {
	Kind   string `yaml:"kind"`
	Script string `yaml:"script"`
}

func (filter *StarlarkFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	snodes := []starlark.Value{}

	for _, rnode := range input {
		snodes = append(snodes, newSNode(rnode))
	}

	slPredeclared := starlark.StringDict{
		"resources": starlark.NewList(snodes),
	}
	slOpts := &syntax.FileOptions{
		TopLevelControl: true,
	}

	slThread := &starlark.Thread{
		Name: "starlark-filter",
		Print: func(thread *starlark.Thread, msg string) {
			slog.Info("starlark filter output", "msg", msg)
		},
	}
	_, err := starlark.ExecFileOptions(
		slOpts,
		slThread,
		"starlark-filter",
		filter.Script,
		slPredeclared,
	)

	return input, err
}

var (
	_ starlark.Value           = (*sNode)(nil)
	_ starlark.Indexable       = (*sNodeSequence)(nil)
	_ starlark.Iterable        = (*sNodeSequence)(nil)
	_ starlark.IterableMapping = (*sNodeMapping)(nil)
	_ starlark.HasSetKey       = (*sNodeMapping)(nil)
	_ starlark.HasAttrs        = (*sNodeMapping)(nil)
	_ starlark.HasSetField     = (*sNodeMapping)(nil)
)

func newSNode(rnode *yaml.RNode) starlark.Value {
	sNode := &sNode{rnode}

	switch rnode.YNode().Kind {
	case yaml.MappingNode:
		return &sNodeMapping{sNode}
	case yaml.SequenceNode:
		return &sNodeSequence{sNode}
	default:
		return sNode
	}
}

type sNode struct {
	*yaml.RNode
}

func (node *sNode) Freeze() {}

func (node *sNode) Type() string {
	switch node.YNode().Kind {
	case yaml.SequenceNode:
		return "SequenceNode"
	case yaml.MappingNode:
		return "MappingNode"
	case yaml.ScalarNode:
		return "ScalarNode"
	case yaml.DocumentNode:
		return "DocumentNode"
	case yaml.AliasNode:
		return "AliasNode"
	default:
		return "Unknown"
	}
}

func (node *sNode) String() string {
	text, _ := node.RNode.String()
	return strings.TrimSpace(text)
}

func (node *sNode) Hash() (uint32, error) {
	return (starlark.String)(node.String()).Hash()
}

func (node *sNode) Truth() starlark.Bool {
	return (starlark.String)(node.String()).Truth()
}

type sNodeSequence struct {
	*sNode
}

func (node *sNodeSequence) Len() int {
	return len(node.Content())
}

func (node *sNodeSequence) Index(idx int) starlark.Value {
	return newSNode(yaml.NewRNode(node.Content()[idx]))
}

func (node *sNodeSequence) Iterate() starlark.Iterator {
	items := []starlark.Value{}

	_ = node.VisitElements(func(node *yaml.RNode) error {
		items = append(items, newSNode(node))
		return nil
	})

	return starlark.NewList(items).Iterate()
}

type sNodeMapping struct {
	*sNode
}

func (node *sNodeMapping) match(path types.NodePath, create yaml.Kind) (*yaml.RNode, error) {
	matcher := &yaml.PathMatcher{
		Path:   path,
		Create: create,
	}

	return matcher.Filter(node.RNode)
}

func (node *sNodeMapping) Get(key starlark.Value) (starlark.Value, bool, error) {
	path, err := slToPath(key)
	if err != nil {
		return nil, false, err
	}

	matches, err := node.match(path, 0)
	if err != nil {
		return nil, false, err
	}

	if matches.IsNilOrEmpty() {
		return starlark.NewList(nil), true, nil
	}

	values := []starlark.Value{}

	err = matches.VisitElements(func(node *yaml.RNode) error {
		values = append(values, newSNode(node))
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	if !path.IsLookup() {
		return values[0], true, nil
	}

	return starlark.NewList(values), true, nil
}

func (node *sNodeMapping) Iterate() starlark.Iterator {
	items := node.Items()
	values := make([]starlark.Value, 0, len(items))

	for _, item := range items {
		values = append(values, item)
	}

	return starlark.NewList(values).Iterate()
}

func (node *sNodeMapping) Items() []starlark.Tuple {
	content := node.Content()
	tuples := make([]starlark.Tuple, 0, len(content)/2)

	for idx := 0; idx < len(content); idx += 2 {
		key, value := content[idx], content[idx+1]
		tuples = append(tuples, starlark.Tuple{
			starlark.String(strings.TrimSpace(key.Value)),
			newSNode(yaml.NewRNode(value)),
		})
	}

	return tuples
}

func (node *sNodeMapping) SetKey(k, v starlark.Value) error {
	return nil
}

func (node *sNodeMapping) AttrNames() []string {
	names, _ := node.Fields()
	return names
}

func (node *sNodeMapping) Attr(name string) (starlark.Value, error) {
	kv := node.Field(name)
	if kv.IsNilOrEmpty() {
		return nil, starlark.NoSuchAttrError(name)
	}

	return newSNode(kv.Value), nil
}

func (node *sNodeMapping) SetField(name string, value starlark.Value) error {
	rnode, err := yaml.Parse(strings.TrimSpace(value.String()))
	if err != nil {
		return err
	}

	return node.PipeE(yaml.SetField(name, rnode))
}

func slToPath(key starlark.Value) (types.NodePath, error) {
	var path types.NodePath

	keyBytes := []byte(key.String())
	if err := yaml.Unmarshal(keyBytes, &path); err != nil {
		return nil, err
	}

	return path, nil
}
