package config

import (
	_ "embed"
	"fmt"

	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const DefaultFileName = "rekustomization.yaml"

//go:embed defaults.yaml
var defaultsYaml []byte

type Rekustomization struct {
	Source Source `yaml:"source"`
	Output Output `yaml:"output"`

	Filters []filters.KFilter `yaml:"filters"`
}

type rekustomization Rekustomization

func (cfg *Rekustomization) UnmarshalYAML(node *yaml.Node) error {
	defaults := &rekustomization{}
	if err := yaml.Unmarshal(defaultsYaml, &defaults); err != nil {
		panic(fmt.Errorf("broken defaults: %w", err))
	}

	base := &rekustomization{}
	if err := node.Decode(base); err != nil {
		return fmt.Errorf("unable to parse config: %w", err)
	}

	cfg.Source = base.Source
	cfg.Output = base.Output
	cfg.Filters = base.Filters
	cfg.setDefaults(defaults)

	return nil
}

func (opts *Rekustomization) setDefaults(defaults *rekustomization) {
	if len(opts.Source.Resources) == 0 && opts.Source.Kustomization == "" {
		opts.Source.Resources = []types.ResourceSelector{{}}
	}

	labelSelectors := defaults.Source.Resources[0].LabelSelectors
	excludeResources := defaults.Source.Resources[0].Resources.Exclude

	for i := range opts.Source.Resources {
		if len(opts.Source.Resources[i].Resources.Include) == 0 {
			opts.Source.Resources[i].Resources.Exclude = append(opts.Source.Resources[i].Resources.Exclude, excludeResources...)
		}

		opts.Source.Resources[i].LabelSelectors = append(opts.Source.Resources[i].LabelSelectors, labelSelectors...)
	}

	opts.Filters = append(opts.Filters, defaults.Filters...)
}
