package filters

import (
	"fmt"
	"log/slog"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/kstar"
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
	output := []*yaml.RNode{}
	schemas := kstar.NewSchemaIndex(nil)

	slPredeclared := starlark.StringDict{
		"resources": kstar.FromRNodes(schemas, input),
		"output":    starlark.NewList(nil),
		"args":      kstar.FromYNode(filter.args.YNode()),
	}
	slOpts := &syntax.FileOptions{
		TopLevelControl: true,
		GlobalReassign: true,
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

	slOutput, ok := slPredeclared["output"].(starlark.Iterable)
	if !ok {
		return nil, fmt.Errorf("starlark filter returned unsupported result")
	}

	var value starlark.Value
	iter := slOutput.Iterate()
	defer iter.Done()

	for iter.Next(&value) {
		ynode, err := kstar.FromStarlark(value)
		if err != nil {
			return nil, err
		}

		output = append(output, yaml.NewRNode(ynode))
	}

	return output, err
}
