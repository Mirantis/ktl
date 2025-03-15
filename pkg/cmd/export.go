package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"

	_ "github.com/Mirantis/rekustomize/pkg/filters" // register filters
	"github.com/Mirantis/rekustomize/pkg/helm"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/kustomize"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
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

			if err := opts.parseClusterFilter(); err != nil {
				return err
			}

			for i := range opts.Filters {
				opts.filters = append(opts.filters, opts.Filters[i].Filter)
			}

			return opts.Run(dir)
		},
	}

	return export
}

type exportOpts struct {
	types.Rekustomization
	clusters      []string
	clustersIndex *types.ClusterIndex
	filters       []kio.Filter
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

func (opts *exportOpts) parseClusterFilter() error {
	if len(opts.Clusters) == 0 {
		return nil
	}

	allClusters, err := kubectl.DefaultCmd().Clusters()
	if err != nil {
		return fmt.Errorf("invalid cluster filter: %w", err)
	}

	opts.clustersIndex = types.BuildClusterIndex(allClusters, opts.Clusters)
	opts.clusters = slices.Collect(opts.clustersIndex.Names(opts.clustersIndex.IDs()...))

	return nil
}

func (opts *exportOpts) Run(dir string) error {
	if len(opts.clusters) > 1 {
		return opts.runMulti(dir)
	}

	return opts.runSingle(dir)
}

func (opts *exportOpts) runMulti(dir string) error {
	waitGroup := &sync.WaitGroup{}
	buffers := map[string]*kio.PackageBuffer{}
	errs := []error{}

	for _, cluster := range opts.clusters {
		buf := &kio.PackageBuffer{}
		buffers[cluster] = buf

		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			kctl := kubectl.DefaultCmd().Cluster(cluster)
			exporter := kubectl.Export{
				Client:    kctl,
				Cluster:   cluster,
				Resources: opts.Resources,
			}

			err := exporter.Execute(buf, opts.filters...)
			errs = append(errs, err)
		}()
	}

	waitGroup.Wait()

	if err := errors.Join(errs...); err != nil {
		return err
	}

	if opts.HelmChart.Name != "" {
		return opts.exportCharts(buffers, dir)
	}

	return opts.exportComponents(buffers, dir)
}

type clusterBuffers = map[string]*kio.PackageBuffer

type clusterResources = map[resid.ResId]map[types.ClusterID]*yaml.RNode

func (opts *exportOpts) convertBuffers(buffers clusterBuffers) (clusterResources, error) {
	resources := map[resid.ResId]map[types.ClusterID]*yaml.RNode{}

	for clusterName, buffer := range buffers {
		cluster, err := opts.clustersIndex.ID(clusterName)
		if err != nil {
			return nil, fmt.Errorf("invalid cluster name: %w", err)
		}

		for _, resNode := range buffer.Nodes {
			id := resid.FromRNode(resNode)

			byCluster, exists := resources[id]
			if !exists {
				byCluster = map[types.ClusterID]*yaml.RNode{}
				resources[id] = byCluster
			}

			byCluster[cluster] = resNode
		}
	}

	return resources, nil
}

func (opts *exportOpts) storeChartOverlays(dir, chartDir string, chart *helm.Chart) error {
	for clusterID, cluster := range opts.clustersIndex.All() {
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

func (opts *exportOpts) exportCharts(buffers map[string]*kio.PackageBuffer, dir string) error {
	chartMeta := opts.HelmChart
	chart := helm.NewChart(chartMeta, opts.clustersIndex)
	chartDir := filepath.Join(dir, "charts", chartMeta.Name)

	if err := os.MkdirAll(chartDir, dirPerm); err != nil {
		return fmt.Errorf("unable to create %v: %w", chartDir, err)
	}

	resources, err := opts.convertBuffers(buffers)
	if err != nil {
		return err
	}

	for id, byCluster := range resources {
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
	for id, cluster := range opts.clustersIndex.All() {
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

func (opts *exportOpts) exportComponents(buffers map[string]*kio.PackageBuffer, dir string) error {
	comps := kustomize.NewComponents(opts.clustersIndex)
	compsDir := filepath.Join(dir, "components")

	resources, err := opts.convertBuffers(buffers)
	if err != nil {
		return err
	}

	for id, byCluster := range resources {
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
	kctl := kubectl.DefaultCmd()

	cluster := "<current-context>"
	if len(opts.clusters) == 1 {
		cluster = opts.clusters[0]
		kctl = kctl.Cluster(cluster)
	}

	out := &kio.LocalPackageWriter{
		PackagePath: dir,
		FileSystem:  filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
	}

	exporter := kubectl.Export{
		Client:    kctl,
		Cluster:   cluster,
		Resources: opts.Resources,
	}

	if err := exporter.Execute(out, opts.filters...); err != nil {
		return fmt.Errorf("unable to export resources: %w", err)
	}
	// REVISIT: overlaps with dedup.Component.Save()
	kust := &types.Kustomization{}
	walkFunc := func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}

		resPath, pathErr := filepath.Rel(dir, path)
		if pathErr != nil {
			return fmt.Errorf("invalid path: %w", pathErr)
		}

		if resPath == types.DefaultFileName {
			return nil
		}

		kust.Resources = append(kust.Resources, resPath)

		return nil
	}

	err := filepath.Walk(dir, walkFunc)
	if err != nil {
		return fmt.Errorf("unable to scan folder: %w", err)
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
