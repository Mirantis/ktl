package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Mirantis/rekustomize/pkg/config"
	_ "github.com/Mirantis/rekustomize/pkg/filters" // register filters
	"github.com/Mirantis/rekustomize/pkg/helm"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/kustomize"
	"github.com/Mirantis/rekustomize/pkg/resource"
	"github.com/Mirantis/rekustomize/pkg/source"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//go:embed defaults.yaml
var defaultsYaml []byte

const (
	dirPerm = 0o700
)

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
	chartDir  string
	compsDir  string
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

func (opts *exportOpts) loadClusterResources() error {
	kubeconfig, err := source.NewKubeconfig(opts.kctl, opts.Source.Clusters)
	if err != nil {
		return err //nolint:wrapcheck
	}

	resources, err := kubeconfig.Resources(opts.Source.Resources, opts.filters)
	if err != nil {
		return err //nolint:wrapcheck
	}

	opts.resources = resources

	return nil
}

func (opts *exportOpts) loadKustomizationResources() error {
	fileSys := &filesys.FileSystemOrOnDisk{}

	path := opts.Source.Kustomization
	if !filepath.IsAbs(path) {
		path = filepath.Clean(filepath.Join(opts.dir, path))
	}

	src, err := source.NewKustomize(
		opts.kctl,
		fileSys,
		path,
		opts.Source.Clusters,
	)
	if err != nil {
		return err //nolint:wrapcheck
	}

	resources, err := src.Resources(opts.Source.Resources, opts.filters)
	if err != nil {
		return err //nolint:wrapcheck
	}

	opts.resources = resources

	return nil
}

func (opts *exportOpts) loadResources() error {
	if opts.Source.Kustomization != "" {
		return opts.loadKustomizationResources()
	}

	return opts.loadClusterResources()
}

func (opts *exportOpts) run() error {
	if err := opts.loadResources(); err != nil {
		return err
	}

	if len(opts.resources.Clusters.IDs()) > 1 {
		return opts.runMulti()
	}

	return opts.runSingle()
}

func (opts *exportOpts) runMulti() error {
	if opts.HelmChart.Name != "" {
		return opts.exportCharts()
	}

	return opts.exportComponents()
}

func (opts *exportOpts) storeChartOverlays(chart *helm.Chart) error {
	for clusterID, cluster := range opts.resources.Clusters.All() {
		fileStore := resource.FileStore{
			Dir:        filepath.Join(opts.dir, "overlays", cluster.Name),
			FileSystem: filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
		}
		kust := &types.Kustomization{}
		kust.Kind = types.KustomizationKind

		chartHome, err := filepath.Rel(fileStore.Dir, filepath.Dir(opts.chartDir))
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		kust.HelmCharts = []types.HelmChart{chart.Instance(clusterID)}
		kust.HelmGlobals = &types.HelmGlobals{
			ChartHome: chartHome,
		}

		if err := fileStore.WriteKustomization(kust); err != nil {
			return fmt.Errorf("unable to store kustomization: %w", err)
		}
	}

	return nil
}

func (opts *exportOpts) exportCharts() error {
	chartMeta := opts.HelmChart
	chart := helm.NewChart(chartMeta, opts.resources.Clusters)
	opts.chartDir = filepath.Join(opts.dir, "charts", chartMeta.Name)

	if err := os.MkdirAll(opts.chartDir, dirPerm); err != nil {
		return fmt.Errorf("unable to create %v: %w", opts.chartDir, err)
	}

	for id, byCluster := range opts.resources.Resources {
		if err := chart.Add(id, byCluster); err != nil {
			return fmt.Errorf("unable to add resources to the chart: %w", err)
		}
	}

	diskFs := filesys.MakeFsOnDisk()
	if err := chart.Store(diskFs, opts.chartDir); err != nil {
		return fmt.Errorf("unable to store the chart: %w", err)
	}

	return opts.storeChartOverlays(chart)
}

func (opts *exportOpts) storeComponentsOverlays(comps *kustomize.Components) error {
	for clusterID, cluster := range opts.resources.Clusters.All() {
		fileStore := resource.FileStore{
			Dir:        filepath.Join(opts.dir, "overlays", cluster.Name),
			FileSystem: filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
		}
		kust := &types.Kustomization{}
		kust.Kind = types.KustomizationKind

		compNames, err := comps.Cluster(clusterID)
		if err != nil {
			panic(err)
		}

		for _, compName := range compNames {
			relPath, err := filepath.Rel(fileStore.Dir, filepath.Join(opts.compsDir, compName))
			if err != nil {
				panic(err)
			}

			kust.Components = append(kust.Components, relPath)
		}

		if err := fileStore.WriteKustomization(kust); err != nil {
			return fmt.Errorf("unable to store kustomization: %w", err)
		}
	}

	return nil
}

func (opts *exportOpts) exportComponents() error {
	comps := kustomize.NewComponents(opts.resources.Clusters)
	opts.compsDir = filepath.Join(opts.dir, "components")

	for id, byCluster := range opts.resources.Resources {
		if err := comps.Add(id, byCluster); err != nil {
			return fmt.Errorf("unable to add resources to the component: %w", err)
		}
	}

	diskFs := filesys.MakeFsOnDisk()
	if err := comps.Store(diskFs, opts.compsDir); err != nil {
		return fmt.Errorf("unable to store components: %w", err)
	}

	return opts.storeComponentsOverlays(comps)
}

func (opts *exportOpts) runSingle() error {
	kust := &types.Kustomization{}
	resourceStore := &resource.FileStore{
		Dir:           opts.dir,
		FileSystem:    filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
		NameGenerator: resource.FileName,
		PostProcessor: func(path string, body []byte) []byte {
			relPath, err := filepath.Rel(opts.dir, path)
			if err != nil {
				panic(err)
			}
			kust.Resources = append(kust.Resources, relPath)

			return body
		},
	}

	if err := resourceStore.WriteAll(opts.resources.All()); err != nil {
		return fmt.Errorf("unable to store files: %w", err)
	}

	slices.Sort(kust.Resources)

	if err := resourceStore.WriteKustomization(kust); err != nil {
		return fmt.Errorf("unable to store kustomization: %w", err)
	}

	return nil
}
