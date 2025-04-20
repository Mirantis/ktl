package runner

import (
	_ "embed"
	"fmt"

	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const DefaultFileName = "rekustomization.yaml"

//go:embed defaults.yaml
var defaultsYaml []byte

type Pipeline struct {
	Source Source `yaml:"source"`
	Output Output `yaml:"output"`

	Filters []filters.KFilter `yaml:"filters"`
}

type rekustomization Pipeline

func (cfg *Pipeline) UnmarshalYAML(node *yaml.Node) error {
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

func (cfg *Pipeline) setDefaults(defaults *rekustomization) {
	if len(cfg.Source.Resources) == 0 && cfg.Source.Kustomization == "" {
		cfg.Source.Resources = []types.ResourceSelector{{}}
	}

	labelSelectors := defaults.Source.Resources[0].LabelSelectors
	excludeResources := defaults.Source.Resources[0].Resources.Exclude

	for i := range cfg.Source.Resources {
		if len(cfg.Source.Resources[i].Resources.Include) == 0 {
			cfg.Source.Resources[i].Resources.Exclude = append(cfg.Source.Resources[i].Resources.Exclude, excludeResources...)
		}

		cfg.Source.Resources[i].LabelSelectors = append(cfg.Source.Resources[i].LabelSelectors, labelSelectors...)
	}

	cfg.Filters = append(cfg.Filters, defaults.Filters...)
}

func (cfg *Pipeline) Run(env *Env) error {
	cfg.Source.Cmd = env.Cmd
	cfg.Source.WorkDir = env.WorkDir
	cfg.Source.FileSys = env.FileSys
	cfg.Output.FileSys = env.FileSys
	cfg.Output.WorkDir = env.WorkDir

	filters := []kio.Filter{}

	for i := range cfg.Filters {
		filters = append(filters, cfg.Filters[i].Filter)
	}

	resources, err := cfg.Source.ClusterResources(filters)
	if err != nil {
		return err
	}

	return cfg.Output.Store(resources)
}
