package cmd

import (
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
	"github.com/Mirantis/rekustomize/pkg/dedup"
	"github.com/Mirantis/rekustomize/pkg/export"
	"github.com/Mirantis/rekustomize/pkg/filter"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	defaultNsResFilter = []string{
		"!*.coordination.k8s.io",
		"!*.discovery.k8s.io",
		"!*.events.k8s.io",
		"!csistoragecapacities.storage.k8s.io",
		"!endpoints",
		"!events",
		"!limitranges",
		"!pods",
		"!replicasets.apps",
	}
	defaultClusterResFilter = []string{
		"!*.admissionregistration.k8s.io",
		"!*.apiregistration.k8s.io",
		"!*.flowcontrol.apiserver.k8s.io",
		"!*.scheduling.k8s.io",
		"!componentstatuses",
		"!csinodes.storage.k8s.io",
		"!nodes",
		"!persistentvolumes",
		"!volumeattachments.storage.k8s.io",
	}
	defaultLabelSelectors = []string{
		"!kubernetes.io/bootstrapping",
	}
	defaultSkipRulesYaml = `
- fields:
  - "status"
  - "metadata.uid"
  - "metadata.selfLink"
  - "metadata.resourceVersion"
  - "metadata.generation"
  - "metadata.creationTimestamp"
  - "metadata.annotations.[kubectl.kubernetes.io/last-applied-configuration]"
- resources:
  - kind: Deployment
    group: apps
    version: v1
  fields:
  - "metadata.annotations.[deployment.kubernetes.io/revision]"
- resources:
  - kind: Service
    version: v1
  fields:
  - "spec.clusterIP"
  - "spec.clusterIPs"
- resources:
  - kind: ConfigMap
    version: v1
    name: "kube-root-ca.crt"
  - kind: ServiceAccount
    version: v1
    name: default
  - kind: ConfigMap
    version: v1
    name: kubernetes
    namespace: default
  - kind: Endpoints
    version: v1
    name: kubernetes
    namespace: default
  - kind: Service
    version: v1
    name: kubernetes
    namespace: default
`
)

func exportCommand() *cobra.Command {
	opts := &exportOpts{}

	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			defaultSkipRules := []types.SkipRule{}
			if err := yaml.Unmarshal([]byte(defaultSkipRulesYaml), &defaultSkipRules); err != nil {
				return fmt.Errorf("broken defaultSkipRules: %v", err)
			}
			dir := args[0]
			cfgData, err := os.ReadFile(filepath.Join(dir, "rekustomization.yaml"))
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			if err := yaml.Unmarshal(cfgData, &opts.Rekustomization); err != nil {
				return err
			}
			opts.NamespacedResources = append(opts.NamespacedResources, defaultNsResFilter...)
			opts.ClusterResources = append(opts.ClusterResources, defaultClusterResFilter...)
			opts.LabelSelectors = append(opts.LabelSelectors, defaultLabelSelectors...)
			opts.SkipRules = append(opts.SkipRules, defaultSkipRules...)
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
}

func (opts *exportOpts) parseCleanupRules() error {
	for _, ruleCfg := range opts.SkipRules {
		rule, err := cleanup.NewSkipRule(
			ruleCfg.Resources,
			ruleCfg.Excluding,
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
				Client:           kctl,
				Name:             cluster,
				NsFilter:         opts.Namespaces,
				NsResFilter:      opts.NamespacedResources,
				ClusterResFilter: opts.ClusterResources,
				Selectors:        opts.LabelSelectors,
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

func (opts *exportOpts) exportCharts(buffers map[string]*kio.PackageBuffer, dir string) error {
	chart, err := dedup.BuildHelmChart(&opts.HelmCharts[0], buffers, opts.clusterGroups, filepath.Join(dir, "charts"))
	if err != nil {
		return err
	}

	diskFs := filesys.MakeFsOnDisk()
	if err := chart.Save(diskFs); err != nil {
		return err
	}
	err = dedup.SaveClusterCharts(diskFs, filepath.Join(dir, "overlays"), chart)
	if err != nil {
		return err
	}

	return nil
}

func (opts *exportOpts) exportComponents(buffers map[string]*kio.PackageBuffer, dir string) error {
	components, err := dedup.Components(buffers, opts.clusterGroups, filepath.Join(dir, "components"))
	if err != nil {
		return err
	}

	diskFs := filesys.MakeFsOnDisk()
	for _, comp := range components {
		if err := comp.Save(diskFs); err != nil {
			return err
		}
	}
	err = dedup.SaveClusters(diskFs, filepath.Join(dir, "overlays"), components)
	if err != nil {
		return err
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
		Client:           kctl,
		Name:             cluster,
		NsFilter:         opts.Namespaces,
		NsResFilter:      opts.NamespacedResources,
		ClusterResFilter: opts.ClusterResources,
		Selectors:        opts.LabelSelectors,
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
