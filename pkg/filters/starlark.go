package filters

import (
	"fmt"
	"iter"
	"log/slog"
	"strings"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/resource"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//nolint:gochecknoinits
func init() {
	filters.Filters["Starlark"] = func() kio.Filter { return &StarlarkFilter{} }
}

func newStarlarkFilter(spec *apis.StarlarkFilter, args *yaml.RNode) (*StarlarkFilter, error) {
	return &StarlarkFilter{
		Kind:   "Starlark",
		Script: spec.GetScript(),
		args:   args,
	}, nil
}

type StarlarkFilter struct {
	Kind   string `yaml:"kind"`
	Script string `yaml:"script"`
	args   *yaml.RNode
}

func (filter *StarlarkFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	const (
		resourcesKey = "resources"
		outputKey    = "output"
		argsKey      = "args"
	)
	snodes := []starlark.Value{}
	output := []*yaml.RNode{}

	for _, rnode := range input {
		snodes = append(snodes, newSNode(rnode))
	}

	slPredeclared := starlark.StringDict{
		resourcesKey: starlark.NewList(snodes),
		outputKey:    starlark.NewList(nil),
		argsKey:      newSNode(filter.args),
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

	slOutput, ok := slPredeclared[outputKey].(starlark.Iterable)
	if !ok {
		return nil, fmt.Errorf("starlark filter returned unsupported %q", outputKey)
	}

	for slItem := range slListAll(slOutput) {
		switch item := slItem.(type) {
		case *sNodeMapping:
			output = append(output, item.RNode)
		case *sNodeSequence:
			output = append(output, item.RNode)
		case *sNode:
			output = append(output, item.RNode)
		default:
			parsed, err := yaml.Parse(item.String())
			if err != nil {
				return nil, fmt.Errorf("starlark result parsing error: %w", err)
			}
			parsed.YNode().Style = 0
			output = append(output, parsed)
		}
	}

	return output, err
}

func slListAll(input starlark.Iterable) iter.Seq[starlark.Value] {
	return func(yield func(starlark.Value) bool) {
		var value starlark.Value
		it := input.Iterate()
		defer it.Done()

		for it.Next(&value) {
			if !yield(value) {
				return
			}
		}
	}
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
	if rnode == nil {
		rnode = yaml.MakeNullNode()
	}

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

func (node *sNodeMapping) match(path resource.Query, create yaml.Kind) (*yaml.RNode, error) {
	matcher := &yaml.PathMatcher{
		Path:   path,
		Create: create,
	}

	return matcher.Filter(node.RNode)
}

func (node *sNodeMapping) Get(key starlark.Value) (starlark.Value, bool, error) {
	path, err := slToQuery(key)
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

func slToQuery(key starlark.Value) (resource.Query, error) {
	var path resource.Query

	keyBytes := []byte(key.String())
	if err := yaml.Unmarshal(keyBytes, &path); err != nil {
		return nil, err
	}

	return path, nil
}
