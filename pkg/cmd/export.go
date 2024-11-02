package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/Mirantis/rekustomize/pkg/dedup"
	"github.com/Mirantis/rekustomize/pkg/export"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/types"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(args[0])
		},
	}
	export.Flags().StringSliceVarP(&opts.nsFilter, "namespace-filter", "n", nil, "TODO: usage")
	export.Flags().StringSliceVarP(&opts.clusters, "clusters", "c", nil, "TODO: usage")
	return export
}

type exportOpts struct {
	nsFilter []string
	clusters []string
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
			err := export.Cluster(kctl, buf, false)
			errs = append(errs, err)
		}()
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return err
	}

	components, err := dedup.Components(buffers)
	if err != nil {
		return err
	}
	clusterComponents := map[string]*types.Kustomization{}
	diskFs := filesys.MakeFsOnDisk()
	for _, comp := range components {
		if err := comp.Save(diskFs, filepath.Join(dir, "components", comp.Name)); err != nil {
			return err
		}
		for _, cluster := range comp.Clusters {
			clusterKust := clusterComponents[cluster]
			if clusterKust == nil {
				clusterKust = &types.Kustomization{}
				clusterKust.Kind = types.KustomizationKind
				clusterComponents[cluster] = clusterKust
			}
			compPath := filepath.Join("..", "..", "components", comp.Name)
			clusterKust.Components = append(clusterKust.Components, compPath)
		}
	}
	for cluster, clusterKust := range clusterComponents {
		data, err := yaml.Marshal(clusterKust)
		if err != nil {
			panic(err)
		}
		kustPath := filepath.Join(dir, "overlays", cluster, "kustomization.yaml")
		os.MkdirAll(filepath.Dir(kustPath), 0o755)
		if err := os.WriteFile(kustPath, data, 0o644); err != nil {
			panic(err)
		}
	}

	return nil
}

func (opts *exportOpts) runSingle(dir string) error {
	kctl := kubectl.DefaultCmd()
	out := &kio.LocalPackageWriter{
		PackagePath: dir,
		FileSystem:  filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
	}

	return export.Cluster(kctl, out, true)
}
