package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	_ "github.com/Mirantis/rekustomize/pkg/filters" // register filters
	"github.com/Mirantis/rekustomize/pkg/helm"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/kustomize"
	"github.com/Mirantis/rekustomize/pkg/resource"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//go:embed defaults.yaml
var defaultsYaml []byte

const (
	dirPerm  = 0o700
	filePerm = 0o600
)

func exportCommand() *cobra.Command {
	opts := &exportOpts{}

	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:revive
			defaults := &types.Rekustomization{}
			if err := yaml.Unmarshal(defaultsYaml, &defaults); err != nil {
				panic(fmt.Errorf("broken defaultSkipRules: %w", err))
			}
			dir := args[0]

			cfgData, err := os.ReadFile(filepath.Join(dir, "rekustomization.yaml"))
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

			return opts.run(dir)
		},
	}

	return export
}

type exportOpts struct {
	types.Rekustomization
	kctl      *kubectl.Cmd
	resources *types.ClusterResources
	filters   []kio.Filter
}

func (opts *exportOpts) setDefaults(defaults *types.Rekustomization) {
	if len(opts.Resources) == 0 {
		opts.Resources = []types.ResourceSelector{{}}
	}

	labelSelectors := defaults.Resources[0].LabelSelectors
	excludeResources := defaults.Resources[0].Resources.Exclude

	for i := range opts.Resources {
		if len(opts.Resources[i].Resources.Include) == 0 {
			opts.Resources[i].Resources.Exclude = append(opts.Resources[i].Resources.Exclude, excludeResources...)
		}

		opts.Resources[i].LabelSelectors = append(opts.Resources[i].LabelSelectors, labelSelectors...)
	}

	opts.Filters = append(opts.Filters, defaults.Filters...)
}

func (opts *exportOpts) run(dir string) error {
	clusters, err := opts.kctl.Clusters(opts.Clusters)
	if err != nil {
		return err
	}

	resources, err := clusters.Resources(opts.Resources, opts.filters)
	if err != nil {
		return err
	}
	opts.resources = resources

	if len(clusters.IDs()) > 1 {
		return opts.runMulti(dir)
	}

	return opts.runSingle(dir)
}

func (opts *exportOpts) runMulti(dir string) error {
	if opts.HelmChart.Name != "" {
		return opts.exportCharts(dir)
	}

	return opts.exportComponents(dir)
}

func (opts *exportOpts) storeChartOverlays(dir, chartDir string, chart *helm.Chart) error {
	for clusterID, cluster := range opts.resources.Clusters.All() {
		path := filepath.Join(dir, "overlays", cluster.Name, "kustomization.yaml")
		kust := &types.Kustomization{}
		kust.Kind = types.KustomizationKind

		chartHome, err := filepath.Rel(filepath.Dir(path), filepath.Dir(chartDir))
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		kust.HelmCharts = []types.HelmChart{chart.Instance(clusterID)}
		kust.HelmGlobals = &types.HelmGlobals{
			ChartHome: chartHome,
		}

		kustBody, err := yaml.Marshal(kust)
		if err != nil {
			return fmt.Errorf("unable to serialize %v: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
			return fmt.Errorf("unable to create %v: %w", path, err)
		}

		if err := os.WriteFile(path, kustBody, filePerm); err != nil {
			return fmt.Errorf("unable to write %v: %w", path, err)
		}
	}

	return nil
}

func (opts *exportOpts) exportCharts(dir string) error {
	chartMeta := opts.HelmChart
	chart := helm.NewChart(chartMeta, opts.resources.Clusters)
	chartDir := filepath.Join(dir, "charts", chartMeta.Name)

	if err := os.MkdirAll(chartDir, dirPerm); err != nil {
		return fmt.Errorf("unable to create %v: %w", chartDir, err)
	}

	for id, byCluster := range opts.resources.Resources {
		if err := chart.Add(id, byCluster); err != nil {
			return fmt.Errorf("unable to add resources to the chart: %w", err)
		}
	}

	diskFs := filesys.MakeFsOnDisk()
	if err := chart.Store(diskFs, chartDir); err != nil {
		return fmt.Errorf("unable to store the chart: %w", err)
	}

	return opts.storeChartOverlays(dir, chartDir, chart)
}

func (opts *exportOpts) storeComponentsOverlays(dir, compsDir string, comps *kustomize.Components) error {
	for id, cluster := range opts.resources.Clusters.All() {
		path := filepath.Join(dir, "overlays", cluster.Name, "kustomization.yaml")
		kust := &types.Kustomization{}
		kust.Kind = types.KustomizationKind

		compNames, err := comps.Cluster(id)
		if err != nil {
			panic(err)
		}

		for _, compName := range compNames {
			relPath, err := filepath.Rel(filepath.Dir(path), filepath.Join(compsDir, compName))
			if err != nil {
				panic(err)
			}

			kust.Components = append(kust.Components, relPath)
		}

		kustBody, err := yaml.Marshal(kust)
		if err != nil {
			return fmt.Errorf("unable to serialize %v: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
			return fmt.Errorf("unable to create %v: %w", path, err)
		}

		if err := os.WriteFile(path, kustBody, filePerm); err != nil {
			return fmt.Errorf("unable to write %v: %w", path, err)
		}
	}

	return nil
}

func (opts *exportOpts) exportComponents(dir string) error {
	comps := kustomize.NewComponents(opts.resources.Clusters)
	compsDir := filepath.Join(dir, "components")

	for id, byCluster := range opts.resources.Resources {
		if err := comps.Add(id, byCluster); err != nil {
			return fmt.Errorf("unable to add resources to the component: %w", err)
		}
	}

	diskFs := filesys.MakeFsOnDisk()
	if err := comps.Store(diskFs, compsDir); err != nil {
		return fmt.Errorf("unable to store components: %w", err)
	}

	return opts.storeComponentsOverlays(dir, compsDir, comps)
}

func (opts *exportOpts) runSingle(dir string) error {
	kust := &types.Kustomization{}
	resourceStore := &resource.FileStore{
		Dir:           dir,
		FileSystem:    filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
		NameGenerator: resource.FileName,
		PostProcessor: func(path string, body []byte) []byte {
			relPath, err := filepath.Rel(dir, path)
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

	kustBytes, err := yaml.Marshal(kust)
	if err != nil {
		return fmt.Errorf("unable to generate kustomization.yaml: %w", err)
	}

	kustPath := filepath.Join(dir, konfig.DefaultKustomizationFileName())
	if err := os.WriteFile(kustPath, kustBytes, filePerm); err != nil {
		return fmt.Errorf("unable to store kustomization.yaml: %w", err)
	}

	return nil
}
