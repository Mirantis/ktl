package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mirantis/rekustomize/pkg/config"
	_ "github.com/Mirantis/rekustomize/pkg/filters" // register filters
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//go:embed defaults.yaml
var defaultsYaml []byte

func exportCommand() *cobra.Command {
	opts := &exportOpts{}

	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:revive
			defaults := &config.Rekustomization{}
			if err := yaml.Unmarshal(defaultsYaml, &defaults); err != nil {
				panic(fmt.Errorf("broken defaultSkipRules: %w", err))
			}
			opts.dir = args[0]

			cfgData, err := os.ReadFile(filepath.Join(opts.dir, "rekustomization.yaml"))
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("unable to read rekustomization.yaml: %w", err)
			}

			if err := yaml.Unmarshal(cfgData, &opts.Rekustomization); err != nil {
				return fmt.Errorf("unable to parse rekustomization.yaml: %w", err)
			}

			opts.setDefaults(defaults)
			opts.kctl = kubectl.New()

			for i := range opts.Filters {
				opts.filters = append(opts.filters, opts.Filters[i].Filter)
			}

			return opts.run()
		},
	}

	return export
}

type exportOpts struct {
	config.Rekustomization
	dir       string
	kctl      *kubectl.Cmd
	resources *types.ClusterResources
	filters   []kio.Filter
}

func (opts *exportOpts) setDefaults(defaults *config.Rekustomization) {
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

func (opts *exportOpts) loadResources() error {
	opts.Source.Cmd = opts.kctl
	opts.Source.FileSys = filesys.MakeFsOnDisk()
	opts.Source.WorkDir = opts.dir

	resources, err := opts.Source.ClusterResources(opts.filters)
	if err != nil {
		return err
	}

	opts.resources = resources

	return nil
}

func (opts *exportOpts) run() error {
	if err := opts.loadResources(); err != nil {
		return err
	}

	opts.Output.FileSys = opts.Source.FileSys
	opts.Output.WorkDir = opts.Source.WorkDir

	return opts.Output.Store(opts.resources)
}
