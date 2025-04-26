package runner

import (
	"fmt"

	"github.com/Mirantis/rekustomize/pkg/source"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Source struct {
	source.Impl
}

func (src *Source) UnmarshalYAML(node *yaml.Node) error {
	meta := &yaml.TypeMeta{}
	if err := node.Decode(meta); err != nil {
		return err //nolint:wrapcheck
	}

	switch meta.Kind {
	case "":
		fallthrough
	case "KubeConfig":
		impl := &source.Kubeconfig{}
		src.Impl = impl

		return node.Decode(impl) //nolint:wrapcheck
	case "Kustomize":
		impl := &source.Kustomize{}
		src.Impl = impl

		return node.Decode(impl) //nolint:wrapcheck
	default:
		return fmt.Errorf("%w: %s", errUnsupportedKind, meta.Kind)
	}
}
