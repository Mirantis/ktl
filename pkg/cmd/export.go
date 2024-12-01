package cmd

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Mirantis/rekustomize/pkg/dedup"
	"github.com/Mirantis/rekustomize/pkg/export"
	"github.com/Mirantis/rekustomize/pkg/filter"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/types"
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
)

func exportCommand() *cobra.Command {
	opts := &exportOpts{}
	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.nsResFilter = append(opts.nsResFilter, defaultNsResFilter...)
			opts.clusterResFilter = append(opts.clusterResFilter, defaultClusterResFilter...)
			return opts.Run(args[0])
		},
	}
	export.Flags().StringSliceVarP(&opts.nsFilter, "namespaces", "n", nil, "namespace filter (default: current kubeconfig context)")
	export.Flags().StringSliceVarP(&opts.nsResFilter, "namespaced-resources", "r", []string{"*"}, "filter for namespaced resources (default: '*')")
	export.Flags().StringSliceVarP(&opts.clusterResFilter, "cluster-resources", "R", []string{"!*"}, "filter for cluster resources (default: '!*')")
	export.Flags().StringSliceVarP(&opts.clusterFilter, "clusters", "c", nil, "cluster filter (default: current kubeconfig context)")
	return export
}

type exportOpts struct {
	nsFilter         []string
	nsResFilter      []string
	clusterResFilter []string
	clusterFilter    []string
	clusters         []string
	clusterGroups    map[string]sets.String
}

func (opts *exportOpts) parseClusterFilter() error {
	if len(opts.clusterFilter) == 0 {
		return nil
	}
	allClusters, err := kubectl.DefaultCmd().Clusters()
	if err != nil {
		return err
	}

	opts.clusterGroups = make(map[string]sets.String)
	filteredClusters := sets.String{}
	group := ""
	for _, filterPart := range opts.clusterFilter {
		var pattern string
		parts := strings.Split(filterPart, "=")
		if len(parts) == 1 {
			pattern = parts[0]
		} else {
			group = parts[0]
			pattern = parts[1]
		}
		matchingClusters, err := filter.SelectNames(allClusters, []string{pattern})
		if err != nil {
			return err
		}
		filteredClusters.Insert(matchingClusters...)
		if group != "" {
			groupSet, found := opts.clusterGroups[group]
			if !found {
				groupSet = sets.String{}
				opts.clusterGroups[group] = groupSet
			}
			groupSet.Insert(matchingClusters...)
		}
	}
	opts.clusters = slices.Collect(maps.Keys(filteredClusters))
	opts.clusterGroups["all-clusters"] = filteredClusters
	return nil
}

func (opts *exportOpts) Run(dir string) error {
	if err := opts.parseClusterFilter(); err != nil {
		return err
	}
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
			err := export.Cluster(kctl, opts.nsFilter, opts.nsResFilter, opts.clusterResFilter, buf, false)
			errs = append(errs, err)
		}()
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return err
	}

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
	out := &kio.LocalPackageWriter{
		PackagePath: dir,
		FileSystem:  filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
	}

	if err := export.Cluster(kctl, opts.nsFilter, opts.nsResFilter, opts.clusterResFilter, out, true); err != nil {
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
