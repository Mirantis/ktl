package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Mirantis/rekustomize/pkg/cleanup"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/yutil"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const resourceBaseYaml = `
apiVersion: PLACEHOLDER
kind: PLACEHOLDER
metadata:
  name: PLACEHOLDER
`

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
		return opts.runMulti(dir)
	}
	return opts.runSingle(dir)
}

type pathKey = string
type valueKey = string
type clusterKey = string
type resourceEntry struct {
	path  yutil.NodePath
	value *yaml.RNode
	root  *yaml.RNode
}
type component struct {
	types.Kustomization
	clusters  []string
	entries   map[resid.ResId][]*resourceEntry
	resources []*yaml.RNode
	patches   []*yaml.RNode
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
			err := opts.export(kctl, buf, false)
			errs = append(errs, err)
		}()
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return err
	}

	index := map[resid.ResId]map[pathKey]map[valueKey]map[clusterKey]*resourceEntry{}
	for cluster, pkg := range buffers {
		for _, rn := range pkg.Nodes {
			id := resid.FromRNode(rn)
			for path, value := range yutil.Flatten(rn) {
				entry := resourceEntry{path: path, value: value, root: rn}
				byPath, ok := index[id]
				if !ok {
					byPath = make(map[pathKey]map[valueKey]map[clusterKey]*resourceEntry)
					index[id] = byPath
				}
				pathStr := path.String()
				byValue, ok := byPath[pathStr]
				if !ok {
					byValue = make(map[valueKey]map[clusterKey]*resourceEntry)
					byPath[pathStr] = byValue
				}
				valueStr, err := value.String()
				if err != nil {
					return err
				}
				byCluster, ok := byValue[valueStr]
				if !ok {
					byCluster = make(map[clusterKey]*resourceEntry)
					byValue[valueStr] = byCluster
				}
				byCluster[cluster] = &entry
			}
		}
	}

	components := map[string]*component{}

	for id, byPath := range index {
		for _, byValue := range byPath {
			for _, byCluster := range byValue {
				clusters := slices.Sorted(maps.Keys(byCluster))
				compKey := strings.Join(clusters, ",")
				comp := components[compKey]
				if nil == comp {
					comp = &component{
						clusters: clusters,
						entries:  map[resid.ResId][]*resourceEntry{},
					}
					comp.Kind = types.ComponentKind
					components[compKey] = comp
				}
				entry := byCluster[clusters[0]]
				comp.entries[id] = append(comp.entries[id], entry)
			}
		}
	}

	seen := map[resid.ResId]bool{}
	compNames := slices.SortedFunc(maps.Keys(components), func(a, b string) int {
		return -cmp.Compare(len(components[a].clusters), len(components[b].clusters))
	})
	resourceBase, err := yaml.Parse(resourceBaseYaml)
	if err != nil {
		panic(err)
	}
	for _, compName := range compNames {
		comp := components[compName]
		for id, entries := range comp.entries {
			rn := resourceBase.Copy()
			rn.SetApiVersion(id.ApiVersion())
			rn.SetName(id.Name)
			rn.SetKind(id.Kind)
			if id.Namespace != "" {
				rn.SetNamespace(id.Namespace)
				setObjectPath(rn, true)
			} else {
				setObjectPath(rn, false)
			}
			meta, err := rn.GetMeta()
			if err != nil {
				panic(err)
			}
			path := meta.Annotations[kioutil.PathAnnotation]

			slices.SortFunc(entries, func(a, b *resourceEntry) int {
				if a.value == nil && b.value == nil {
					return 0
				}
				if a.value == nil {
					return -1
				}
				if b.value == nil {
					return 1
				}
				ay := a.value.YNode()
				by := b.value.YNode()
				if ay == nil && by == nil {
					return 0
				}
				if ay == nil {
					return -1
				}
				if by == nil {
					return 1
				}
				return cmp.Compare(ay.Line, by.Line)
			})
			for _, entry := range entries {
				if entry.value == nil || entry.value.YNode() == nil {
					continue
				}
				vn, err := rn.Pipe(yaml.LookupCreate(entry.value.YNode().Kind, entry.path...))
				if err != nil {
					panic(err)
				}
				vn.SetYNode(entry.value.YNode())
			}
			// FIXME: duplicate (annotations are cleared with "{}" value)
			if id.Namespace != "" {
				rn.SetNamespace(id.Namespace)
				setObjectPath(rn, true)
			} else {
				setObjectPath(rn, false)
			}

			if seen[id] {
				comp.patches = append(comp.patches, rn)
				comp.Patches = append(comp.Patches, types.Patch{Path: path})
			} else {
				comp.resources = append(comp.resources, rn)
				comp.Resources = append(comp.Resources, path)
				seen[id] = true
			}
		}
	}

	diskFs := filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()}
	clusterComponents := map[string]*types.Kustomization{}
	for _, compName := range compNames {
		comp := components[compName]
		slices.SortFunc(comp.Patches, func(a, b types.Patch) int {
			return cmp.Compare(a.Path, b.Path)
		})
		slices.Sort(comp.Resources)
		compWriter := &kio.LocalPackageWriter{
			PackagePath: filepath.Join(dir, "components", compName),
			FileSystem:  diskFs,
		}
		os.MkdirAll(compWriter.PackagePath, 0o755)
		err := kio.Pipeline{
			Inputs: []kio.Reader{
				&kio.PackageBuffer{Nodes: comp.resources},
				&kio.PackageBuffer{Nodes: comp.patches},
			},
			Outputs: []kio.Writer{compWriter},
		}.Execute()
		if err != nil {
			panic(err)
		}
		kustBytes, err := yaml.Marshal(&comp.Kustomization)
		if err != nil {
			panic(err)
		}
		kustPath := filepath.Join(compWriter.PackagePath, "kustomization.yaml")
		if err := os.WriteFile(kustPath, kustBytes, 0o644); err != nil {
			panic(err)
		}
		for _, cluster := range comp.clusters {
			clusterKust := clusterComponents[cluster]
			if clusterKust == nil {
				clusterKust = &types.Kustomization{}
				clusterKust.Kind = types.KustomizationKind
				clusterComponents[cluster] = clusterKust
			}
			compPath := filepath.Join("..", "..", "components", compName)
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

	return opts.export(kctl, out, true)
}

func (opts *exportOpts) export(kctl kubectl.Cmd, out kio.Writer, setPath bool) error {
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

		if setPath {
			for _, obj := range objects {
				setObjectPath(obj, true)
			}
		}
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
