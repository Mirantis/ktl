package cmd

import (
	"errors"
	"path/filepath"
	"sync"

	"github.com/Mirantis/rekustomize/pkg/dedup"
	"github.com/Mirantis/rekustomize/pkg/export"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

var defaultResourceFilter = []string{
	"!*.admissionregistration.k8s.io",
	"!*.apiregistration.k8s.io",
	"!*.coordination.k8s.io",
	"!*.discovery.k8s.io",
	"!*.events.k8s.io",
	"!*.flowcontrol.apiserver.k8s.io",
	"!*.scheduling.k8s.io",
	"!componentstatuses",
	"!csinodes.storage.k8s.io",
	"!csistoragecapacities.storage.k8s.io",
	"!events",
	"!limitranges",
	"!nodes",
	"!persistentvolumes",
	"!pods",
	"!replicasets.apps",
	"!volumeattachments.storage.k8s.io",
}

func exportCommand() *cobra.Command {
	opts := &exportOpts{}
	export := &cobra.Command{
		Use:   "export PATH",
		Short: "TODO: export (short)",
		Long:  "TODO: export (long)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.resFilter = append(opts.resFilter, defaultResourceFilter...)
			return opts.Run(args[0])
		},
	}
	export.Flags().StringSliceVarP(&opts.nsFilter, "namespace-filter", "n", nil, "TODO: usage")
	export.Flags().StringSliceVarP(&opts.resFilter, "resource-filter", "R", nil, "TODO: usage")
	export.Flags().StringSliceVarP(&opts.clusters, "clusters", "c", nil, "TODO: usage")
	return export
}

type exportOpts struct {
	nsFilter  []string
	resFilter []string
	clusters  []string
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
			err := export.Cluster(kctl, opts.nsFilter, opts.resFilter, buf, false)
			errs = append(errs, err)
		}()
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return err
	}

	components, err := dedup.Components(buffers, filepath.Join(dir, "components"))
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

	return export.Cluster(kctl, opts.nsFilter, opts.resFilter, out, true)
}
