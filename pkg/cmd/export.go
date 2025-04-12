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


func exportCommand() *cobra.Command {
	opts := &exportOpts{}

	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:revive
			opts.dir = args[0]

			cfgData, err := os.ReadFile(filepath.Join(opts.dir, "rekustomization.yaml"))
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("unable to read rekustomization.yaml: %w", err)
			}

			if err := yaml.Unmarshal(cfgData, &opts.Rekustomization); err != nil {
				return fmt.Errorf("unable to parse rekustomization.yaml: %w", err)
			}

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
