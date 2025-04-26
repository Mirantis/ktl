package runner

import (
	"fmt"

	"github.com/Mirantis/rekustomize/pkg/output"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)


type Output struct {
	output.Impl
}

func (out *Output) UnmarshalYAML(node *yaml.Node) error {
	meta := &yaml.TypeMeta{}
	if err := node.Decode(meta); err != nil {
		return err //nolint:wrapcheck
	}

	switch meta.Kind {
	case "":
		fallthrough
	case "Kustomize":
		impl := &output.KustomizeOutput{}
		out.Impl = impl

		return node.Decode(impl) //nolint:wrapcheck
	case "KustomizeComponents":
		impl := &output.ComponentsOutput{}
		out.Impl = impl

		return node.Decode(impl) //nolint:wrapcheck
	case "HelmChart":
		impl := &output.ChartOutput{}
		out.Impl = impl

		return node.Decode(impl) //nolint:wrapcheck
	default:
		return fmt.Errorf("%w: %s", errUnsupportedKind, meta.Kind)
	}
}
