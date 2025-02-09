package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/Mirantis/rekustomize/pkg/cleanup"
	"github.com/Mirantis/rekustomize/pkg/export"
	"github.com/Mirantis/rekustomize/pkg/filter"
	"github.com/Mirantis/rekustomize/pkg/helm"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/kustomize"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	//go:embed defaults.yaml
	defaultsYaml []byte
)

func exportCommand() *cobra.Command {
	opts := &exportOpts{}

	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			defaults := &types.Rekustomization{}
			if err := yaml.Unmarshal(defaultsYaml, &defaults); err != nil {
				panic(fmt.Errorf("broken defaultSkipRules: %w", err))
			}
			dir := args[0]
			cfgData, err := os.ReadFile(filepath.Join(dir, "rekustomization.yaml"))
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if err := yaml.Unmarshal(cfgData, &opts.Rekustomization); err != nil {
				return err
			}
			opts.setDefaults(defaults)
			if err := opts.parseClusterFilter(); err != nil {
				return err
			}
			if err := opts.parseCleanupRules(); err != nil {
				return err
			}
			opts.cleanupRules = append(opts.cleanupRules, cleanup.DefaultRules()...)
			return opts.Run(dir)
		},
	}
	return export
}

type exportOpts struct {
	types.Rekustomization
	clusters      []string
	clusterGroups map[string]sets.String
	cleanupRules  cleanup.Rules
	clustersIndex *types.ClusterIndex
}

func (opts *exportOpts) setDefaults(defaults *types.Rekustomization) {
	if len(opts.ExportRules) == 0 {
		opts.ExportRules = []types.ExportRule{{}}
	}

	labelSelectors := defaults.ExportRules[0].LabelSelectors
	excludeResources := defaults.ExportRules[0].Resources.Exclude
	for i := range opts.ExportRules {
		if len(opts.ExportRules[i].Resources.Include) == 0 {
			opts.ExportRules[i].Resources.Exclude = append(opts.ExportRules[i].Resources.Exclude, excludeResources...)
		}
		opts.ExportRules[i].LabelSelectors = append(opts.ExportRules[i].LabelSelectors, labelSelectors...)
	}
	opts.SkipRules = append(opts.SkipRules, defaults.SkipRules...)
}

func (opts *exportOpts) parseCleanupRules() error {
	for _, ruleCfg := range opts.SkipRules {
		rule, err := cleanup.NewSkipRule(
			ruleCfg.If,
			ruleCfg.IfNot,
			ruleCfg.Fields,
		)
		if err != nil {
			return err
		}
		opts.cleanupRules = append(opts.cleanupRules, rule)
	}
	return nil
}

func (opts *exportOpts) parseClusterFilter() error {
	if len(opts.Clusters) == 0 {
		return nil
	}
	allClusters, err := kubectl.DefaultCmd().Clusters()
	if err != nil {
		return err
	}

	opts.clusterGroups = make(map[string]sets.String)
	filteredClusters := sets.String{}
	for _, group := range opts.Clusters {
		matchingClusters, err := filter.SelectNames(allClusters, group.Names)
		if err != nil {
			return err
		}
		filteredClusters.Insert(matchingClusters...)
		groupSet, found := opts.clusterGroups[group.Group]
		if !found {
			groupSet = sets.String{}
			opts.clusterGroups[group.Group] = groupSet
		}
		groupSet.Insert(matchingClusters...)
	}
	opts.clusters = slices.Collect(maps.Keys(filteredClusters))
	opts.clusterGroups["all-clusters"] = filteredClusters
	skippedClusters := sets.String{}
	skippedClusters.Insert(allClusters...)
	skippedClusters = skippedClusters.Difference(filteredClusters)
	slog.Info(
		"clusters",
		"selected", slices.Sorted(maps.Keys(filteredClusters)),
		"skipped", slices.Sorted(maps.Keys(skippedClusters)),
	)
	opts.clustersIndex = types.NewClusterIndex()
	clusters := map[string]*types.Cluster{}
	for _, group := range slices.Sorted(maps.Keys(opts.clusterGroups)) {
		names := opts.clusterGroups[group]
		for name := range names {
			cluster, found := clusters[name]
			if !found {
				cluster = &types.Cluster{Name: name}
				clusters[name] = cluster
			}
			cluster.Tags = append(cluster.Tags, group)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(clusters)) {
		opts.clustersIndex.Add(*clusters[name])
	}
	return nil
}

func (opts *exportOpts) Run(dir string) error {
	if len(opts.clusters) > 1 {
		return opts.runMulti(dir)
	}
	return opts.runSingle(dir)
}

func (opts *exportOpts) runMulti(dir string) error {
	wg := &sync.WaitGroup{}
	buffers := map[string]*kio.PackageBuffer{}
	errs := []error{}
	for _, cluster := range opts.clusters {
		buf := &kio.PackageBuffer{}
		buffers[cluster] = buf
		wg.Add(1)
		go func() {
			defer wg.Done()
			kctl := kubectl.DefaultCmd().Cluster(cluster)

			exporter := export.Cluster{
				Client: kctl,
				Name:   cluster,
				Rules:  opts.ExportRules,
			}

			err := exporter.Execute(buf, opts.cleanupRules)
			errs = append(errs, err)
		}()
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return err
	}
	if len(opts.HelmCharts) > 0 {
		return opts.exportCharts(buffers, dir)
	}

	return opts.exportComponents(buffers, dir)
}
func (opts *exportOpts) convertBuffers(buffers map[string]*kio.PackageBuffer) (map[resid.ResId]map[types.ClusterId]*yaml.RNode, error) {
	resources := map[resid.ResId]map[types.ClusterId]*yaml.RNode{}
	for clusterName, buffer := range buffers {
		cluster, err := opts.clustersIndex.Id(clusterName)
		if err != nil {
			return nil, err
		}
		for _, rn := range buffer.Nodes {
			id := resid.FromRNode(rn)
			byCluster, exists := resources[id]
			if !exists {
				byCluster = map[types.ClusterId]*yaml.RNode{}
				resources[id] = byCluster
			}
			byCluster[cluster] = rn
		}
	}
	return resources, nil
}

func (opts *exportOpts) exportCharts(buffers map[string]*kio.PackageBuffer, dir string) error {
	chartMeta := opts.HelmCharts[0]
	chart := helm.NewChart(chartMeta, opts.clustersIndex)
	chartDir := filepath.Join(dir, "charts", chartMeta.Name)
	if err := os.MkdirAll(chartDir, 0o777); err != nil {
		return fmt.Errorf("unable to create %v: %w", chartDir, err)
	}

	resources, err := opts.convertBuffers(buffers)
	if err != nil {
		return err
	}

	for id, byCluster := range resources {
		if err := chart.Add(id, byCluster); err != nil {
			return err
		}
	}

	diskFs := filesys.MakeFsOnDisk()
	if err := chart.Store(diskFs, chartDir); err != nil {
		return err
	}
	for id, cluster := range opts.clustersIndex.All() {
		path := filepath.Join(dir, "overlays", cluster.Name, "kustomization.yaml")
		kust := &types.Kustomization{}
		kust.Kind = types.KustomizationKind
		chartHome, err := filepath.Rel(filepath.Dir(path), filepath.Dir(chartDir))
		kust.HelmGlobals = &types.HelmGlobals{
			ChartHome: chartHome,
		}
		kust.HelmCharts = []types.HelmChart{chart.Instance(id)}
		kustBody, err := yaml.Marshal(kust)
		if err != nil {
			return fmt.Errorf("unable to serialize %v: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
			return fmt.Errorf("unable to create %v: %w", path, err)
		}
		if err := os.WriteFile(path, kustBody, 0o666); err != nil {
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
			return err
		}
	}

	diskFs := filesys.MakeFsOnDisk()
	if err := comps.Store(diskFs, compsDir); err != nil {
		return err
	}
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
		if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
			return fmt.Errorf("unable to create %v: %w", path, err)
		}
		if err := os.WriteFile(path, kustBody, 0o666); err != nil {
			return fmt.Errorf("unable to write %v: %w", path, err)
		}
	}

	return nil
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

	exporter := export.Cluster{
		Client: kctl,
		Name:   cluster,
		Rules:  opts.ExportRules,
	}

	if err := exporter.Execute(out, opts.cleanupRules); err != nil {
		return err
	}
	// REVISIT: overlaps with dedup.Component.Save()
	kust := &types.Kustomization{}
	filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		resPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if resPath == types.DefaultFileName {
			return nil
		}
		kust.Resources = append(kust.Resources, resPath)
		return nil
	})
	slices.Sort(kust.Resources)
	kustBytes, err := yaml.Marshal(kust)
	if err != nil {
		return err
	}
	kustPath := filepath.Join(dir, konfig.DefaultKustomizationFileName())
	if err := os.WriteFile(kustPath, kustBytes, 0o644); err != nil {
		return err
	}
	return nil
}
