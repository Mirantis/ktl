package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/cleanup"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
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

func setObjectPath(obj *yaml.RNode, withNamespace bool) {
	annotations := map[string]string{}
	for k, v := range obj.GetAnnotations() {
		annotations[k] = v
	}
	path := fmt.Sprintf(
		"%s-%s.yaml",
		obj.GetName(),
		strings.ToLower(obj.GetKind()),
	)
	ns := obj.GetNamespace()
	if withNamespace && ns != "" {
		path = filepath.Join(ns, path)
	}
	annotations[kioutil.PathAnnotation] = path
	obj.SetAnnotations(annotations)
}

func (opts *exportOpts) Run(dir string) error {
	if len(opts.clusters) > 1 {
		opts.runMulti(dir)
	}
	return opts.runSingle(dir)
}

func (opts *exportOpts) runMulti(dir string) error {
	panic("not implemented")
}

func (opts *exportOpts) runSingle(dir string) error {
	kctl := kubectl.DefaultCmd()
	resources, err := kctl.ApiResources()
	if err != nil {
		return err
	}
	inputs := []kio.Reader{}
	for _, resource := range resources {
		// TODO: add another 'skip' layer before the Get call
		objects, err := kctl.Get(resource)
		if err != nil {
			return err
		}
		inputs = append(inputs, &kio.PackageBuffer{Nodes: objects})

		for _, obj := range objects {
			setObjectPath(obj, true)
		}
	}
	out := &kio.LocalPackageWriter{
		PackagePath: dir,
		FileSystem:  filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
	}

	pipeline := &kio.Pipeline{
		Inputs: inputs,
		Filters: []kio.Filter{
			cleanup.DefaultRules(),
			// REVISIT: FileSetter annotations are ignored - bug in kustomize?
			// 1) cleared if no annotation is set before the filter is applied
			// 2) reverted if path annotations was set before the filter is applied
			// &filters.FileSetter{FilenamePattern: "%n_%k.yaml"},
		},
		Outputs: []kio.Writer{out},
	}

	err = pipeline.Execute()
	return err
}
