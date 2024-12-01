package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/Mirantis/rekustomize/pkg/cleanup"
	"github.com/Mirantis/rekustomize/pkg/dedup"
	"github.com/Mirantis/rekustomize/pkg/export"
	"github.com/Mirantis/rekustomize/pkg/filter"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/yutil"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/sets"
	kyutil "sigs.k8s.io/kustomize/kyaml/utils"
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
)

func exportCommand() *cobra.Command {
	clustersFilters := []string{}
	cleanupRules := []string{}
	opts := &exportOpts{
		cleanupRules: cleanup.DefaultRules(),
	}

	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.nsResFilter = append(opts.nsResFilter, defaultNsResFilter...)
			opts.clusterResFilter = append(opts.clusterResFilter, defaultClusterResFilter...)
			opts.labelSelectors = append(opts.labelSelectors, defaultLabelSelectors...)
			if err := opts.parseClusterFilter(clustersFilters); err != nil {
				return err
			}
			if err := opts.parseCleanupRules(cleanupRules); err != nil {
				return err
			}
			return opts.Run(args[0])
		},
	}
	export.Flags().StringSliceVarP(&opts.nsFilter, "namespaces", "n", nil, "namespace filter (default: current kubeconfig context)")
	export.Flags().StringSliceVarP(&opts.nsResFilter, "namespaced-resources", "r", []string{"*"}, "filter for namespaced resources (default: '*')")
	export.Flags().StringSliceVarP(&opts.clusterResFilter, "cluster-resources", "R", []string{"!*"}, "filter for cluster resources (default: '!*')")
	export.Flags().StringSliceVarP(&clustersFilters, "clusters", "c", nil, "cluster filter (default: current kubeconfig context)")
	export.Flags().StringSliceVar(&cleanupRules, "clear-fields", nil, "list of fields to clear (TODO: detailed explanation or move to config)")
	export.Flags().StringSliceVarP(&opts.labelSelectors, "selector", "l", nil, ("" +
		"Selector (label query) to filter on, " +
		"supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). " +
		"Matching objects must satisfy all of the specified label constraints."))
	return export
}

type exportOpts struct {
	nsFilter         []string
	nsResFilter      []string
	clusterResFilter []string
	clusters         []string
	clusterGroups    map[string]sets.String
	labelSelectors   []string
	cleanupRules     cleanup.Rules
}

func (opts *exportOpts) parseCleanupRules(cleanupRules []string) error {
	for _, rawRule := range cleanupRules {
		rule := &cleanup.RegexpRule{}
		// REVISIT: better syntax / refactor
		parts := strings.SplitN(rawRule, "@", 2)
		rule.Path = yutil.NodePath(kyutil.SmarterPathSplitter(parts[0], "."))
		regexStr := `.*`
		if len(parts) > 1 {
			regexStr = parts[1]
		}
		var err error
		if rule.Regexp, err = regexp.Compile(regexStr); err != nil {
			return fmt.Errorf("unable to parse rule %q: %v", rawRule, err)
		}
		opts.cleanupRules = append(opts.cleanupRules, rule)
	}
	return nil
}

func (opts *exportOpts) parseClusterFilter(clusterFilter []string) error {
	if len(clusterFilter) == 0 {
		return nil
	}
	allClusters, err := kubectl.DefaultCmd().Clusters()
	if err != nil {
		return err
	}

	opts.clusterGroups = make(map[string]sets.String)
	filteredClusters := sets.String{}
	group := ""
	for _, filterPart := range clusterFilter {
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
			err := export.Cluster(cluster, kctl, opts.nsFilter, opts.nsResFilter, opts.clusterResFilter, opts.labelSelectors, opts.cleanupRules, buf, false)
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
	cluster := "<current-context>"
	if len(opts.clusters) == 1 {
		cluster = opts.clusters[0]
		kctl = kctl.Cluster(cluster)
	}
	out := &kio.LocalPackageWriter{
		PackagePath: dir,
		FileSystem:  filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
	}

	if err := export.Cluster(cluster, kctl, opts.nsFilter, opts.nsResFilter, opts.clusterResFilter, opts.labelSelectors, opts.cleanupRules, out, true); err != nil {
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
