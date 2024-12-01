package dedup

import (
	"cmp"
	"crypto/sha1"
	"encoding/hex"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/export"
	"github.com/Mirantis/rekustomize/pkg/yutil"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const resourceBaseYaml = `
apiVersion: PLACEHOLDER
kind: PLACEHOLDER
metadata:
  name: PLACEHOLDER
`

type pathKey = string
type valueKey = string
type clusterKey = string
type resourceEntry struct {
	path  yutil.NodePath
	value *yaml.RNode
	root  *yaml.RNode
}
type Component struct {
	types.Kustomization
	Name          string
	Path          string
	Clusters      []string
	entries       map[resid.ResId][]*resourceEntry
	ResourceNodes []*yaml.RNode
	PatchNodes    []*yaml.RNode
}

func (comp *Component) Save(fSys filesys.FileSystem) error {
	diskFs := filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()}
	compWriter := &kio.LocalPackageWriter{
		PackagePath: comp.Path,
		FileSystem:  diskFs,
	}
	os.MkdirAll(compWriter.PackagePath, 0o755)
	err := kio.Pipeline{
		Inputs: []kio.Reader{
			&kio.PackageBuffer{Nodes: comp.ResourceNodes},
			&kio.PackageBuffer{Nodes: comp.PatchNodes},
		},
		Outputs: []kio.Writer{compWriter},
	}.Execute()
	if err != nil {
		return err
	}
	kustBytes, err := yaml.Marshal(&comp.Kustomization)
	if err != nil {
		return err
	}
	kustPath := filepath.Join(compWriter.PackagePath, konfig.DefaultKustomizationFileName())
	if err := os.WriteFile(kustPath, kustBytes, 0o644); err != nil {
		return err
	}
	return nil
}

func componentName(clusters []string, groups map[string]sets.String) string {
	groupNames := slices.Collect(maps.Keys(groups))
	slices.SortFunc(groupNames, func(a, b string) int {
		if ret := len(groups[b]) - len(groups[a]); ret != 0 {
			return ret
		}
		return cmp.Compare(a, b)
	})
	ungroupedClusters := sets.String{}
	ungroupedClusters.Insert(clusters...)
	parts := []string{}
	for _, groupName := range groupNames {
		groupClusters := groups[groupName]
		if len(groupClusters.Difference(ungroupedClusters)) > 0 {
			continue
		}
		ungroupedClusters = ungroupedClusters.Difference(groupClusters)
		parts = append(parts, groupName)
	}
	parts = append(parts, slices.Collect(maps.Keys(ungroupedClusters))...)
	slices.Sort(parts)
	name := strings.Join(parts, ",")
	if len(name) > 255 {
		nameSha1 := sha1.Sum([]byte(name))
		name = hex.EncodeToString(nameSha1[:])
		slog.Warn("component name shortened", "new-name", name, "clusters", parts)
	}
	return name
}

func Components(buffers map[string]*kio.PackageBuffer, groups map[string]sets.String, dir string) ([]*Component, error) {
	index := map[resid.ResId]map[pathKey]map[valueKey]map[clusterKey]*resourceEntry{}
	for cluster, pkg := range buffers {
		for _, rn := range pkg.Nodes {
			id := resid.FromRNode(rn)
			for path, value := range Flatten(rn) {
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
					return nil, err
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

	components := map[string]*Component{}

	for id, byPath := range index {
		for _, byValue := range byPath {
			for _, byCluster := range byValue {
				clusters := slices.Sorted(maps.Keys(byCluster))
				compKey := componentName(clusters, groups)
				comp := components[compKey]
				if nil == comp {
					comp = &Component{
						Name:     compKey,
						Path:     filepath.Join(dir, compKey),
						Clusters: clusters,
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
		return -cmp.Compare(len(components[a].Clusters), len(components[b].Clusters))
	})
	resourceBase, err := yaml.Parse(resourceBaseYaml)
	if err != nil {
		panic(err)
	}
	result := []*Component{}
	for _, compName := range compNames {
		comp := components[compName]
		result = append(result, comp)
		for id, entries := range comp.entries {
			rn := resourceBase.Copy()
			rn.SetApiVersion(id.ApiVersion())
			rn.SetName(id.Name)
			rn.SetKind(id.Kind)
			if id.Namespace != "" {
				rn.SetNamespace(id.Namespace)
				export.SetObjectPath(rn, true)
			} else {
				export.SetObjectPath(rn, false)
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
				export.SetObjectPath(rn, true)
			} else {
				export.SetObjectPath(rn, false)
			}

			if seen[id] {
				comp.PatchNodes = append(comp.PatchNodes, rn)
				comp.Patches = append(comp.Patches, types.Patch{Path: path})
			} else {
				comp.ResourceNodes = append(comp.ResourceNodes, rn)
				comp.Resources = append(comp.Resources, path)
				seen[id] = true
			}
		}
	}

	for _, comp := range result {
		slices.SortFunc(comp.Patches, func(a, b types.Patch) int {
			return cmp.Compare(a.Path, b.Path)
		})
		slices.Sort(comp.Resources)
	}

	return result, nil
}

func SaveClusters(fSys filesys.FileSystem, dir string, components []*Component) error {
	clusterComponents := map[string]*types.Kustomization{}
	for _, comp := range components {
		for _, cluster := range comp.Clusters {
			kustPath := filepath.Join(dir, cluster, "kustomization.yaml")
			clusterKust := clusterComponents[kustPath]
			if clusterKust == nil {
				clusterKust = &types.Kustomization{}
				clusterKust.Kind = types.KustomizationKind
				clusterComponents[kustPath] = clusterKust
			}
			compPath, err := filepath.Rel(filepath.Dir(kustPath), comp.Path)
			if err != nil {
				return err
			}
			clusterKust.Components = append(clusterKust.Components, compPath)
		}
	}
	for kustPath, clusterKust := range clusterComponents {
		data, err := yaml.Marshal(clusterKust)
		if err != nil {
			panic(err)
		}
		os.MkdirAll(filepath.Dir(kustPath), 0o755)
		if err := os.WriteFile(kustPath, data, 0o644); err != nil {
			panic(err)
		}
	}

	return nil
}
