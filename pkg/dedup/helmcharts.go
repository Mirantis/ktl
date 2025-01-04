package dedup

import (
	"cmp"
	"crypto/sha1"
	_ "embed"
	"encoding/hex"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/export"
	"github.com/Mirantis/rekustomize/pkg/helm"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/Mirantis/rekustomize/pkg/yutil"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//go:embed data/_helpers.tpl
var helmHelpers []byte

type HelmChart struct {
	*types.HelmChart
	Path           string
	entries        map[resid.ResId][]*resourceEntry
	Templates      []*yaml.RNode
	Values         *yaml.RNode
	ClusterPresets map[string]sets.String
	ClusterValues  map[string]*yaml.RNode
	token          string
}

func (chart *HelmChart) Save(fSys filesys.FileSystem) error {
	diskFs := filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()}
	chartWriter := &kio.LocalPackageWriter{
		PackagePath: filepath.Join(chart.Path, "templates"),
		FileSystem:  diskFs,
	}
	if err := os.MkdirAll(chartWriter.PackagePath, 0o755); err != nil {
		return err
	}
	err := kio.Pipeline{
		Inputs: []kio.Reader{
			&kio.PackageBuffer{Nodes: chart.Templates},
		},
		Outputs: []kio.Writer{chartWriter},
	}.Execute()
	if err != nil {
		return err
	}
	values, err := chart.Values.String()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(chart.Path, "values.yaml"), []byte(values), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(chart.Path, "templates", "_helpers.tpl"), helmHelpers, 0o644); err != nil {
		return err
	}
	chartMeta := map[string]string{
		"apiVersion": "v2",
		"name":       chart.Name,
		"version":    chart.Version,
	}
	chartBytes, err := yaml.Marshal(chartMeta)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(chart.Path, "Chart.yaml"), chartBytes, 0o644); err != nil {
		return err
	}
	err = chartWriter.FileSystem.Walk(
		chartWriter.PackagePath,
		func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".yaml") {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				newData := []byte(strings.ReplaceAll(string(data), "# HELM"+chart.token+": ", ""))
				if err := os.WriteFile(path, newData, info.Mode().Perm()); err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return err
	}
	return nil
}

func presetName(clusters []string, groups map[string]sets.String) string {
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

func resourceKey(id resid.ResId) string {
	resKey := id.Kind + "/" + id.Name
	if id.Namespace != "" {
		resKey = id.Namespace + "/" + resKey
	}
	return resKey
}

func BuildHelmChart(cfg *types.HelmChart, buffers map[string]*kio.PackageBuffer, groups map[string]sets.String, dir string) (*HelmChart, error) {
	chart := &HelmChart{
		HelmChart:      cfg,
		Values:         yaml.NewMapRNode(&map[string]string{}),
		ClusterPresets: map[string]sets.String{},
		ClusterValues:  map[string]*yaml.RNode{},
		Path:           filepath.Join(dir, cfg.Name),
		entries:        map[resid.ResId][]*resourceEntry{},
		token:          rand.String(8),
	}
	chart.Values.PipeE(yaml.LookupCreate(yaml.MappingNode, "global"))
	presets, _ := chart.Values.Pipe(yaml.LookupCreate(yaml.MappingNode, "preset_values"))
	index := map[resid.ResId]map[pathKey]map[valueKey]map[clusterKey]*resourceEntry{}
	for cluster, pkg := range buffers {
		chart.ClusterPresets[cluster] = sets.String{}
		chart.ClusterValues[cluster] = yaml.NewMapRNode(&map[string]string{})
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

	keyCount := map[string]int{}
	for id, byPath := range index {
		for pathKey, byValue := range byPath {
			count := 0
			valueKey := resourceKey(id) + pathKey
			key := path.Dir(valueKey)
			for _, byCluster := range byValue {
				count += len(byCluster)
			}
			keyCount[key] = max(keyCount[key], count)
		}
	}

	allPresets := sets.String{}
	resPresets := map[string]sets.String{}
	for id, byPath := range index {
		resKey := resourceKey(id)
		resPresets[resKey] = sets.String{}
		for pathKey, byValue := range byPath {
			var entry *resourceEntry
			valueKey := resKey + pathKey
			clustersCount := 0

			for _, byCluster := range byValue {
				clusters := slices.Sorted(maps.Keys(byCluster))
				clustersCount += len(clusters)
				presetKey := presetName(clusters, groups)
				entry = byCluster[clusters[0]]
				resPresets[resKey].Insert(presetKey)
				allPresets.Insert(presetKey)
				if len(byValue) == 1 && clustersCount == keyCount[path.Dir(valueKey)] {
					continue
				}
				var presetValues *yaml.RNode
				if _, isCluster := buffers[presetKey]; isCluster {
					presetValues = chart.ClusterValues[presetKey]
				} else {
					var err error
					presetValues, err = presets.Pipe(yaml.LookupCreate(yaml.MappingNode, presetKey))
					if err != nil {
						panic(err)
					}
				}
				vn, err := presetValues.Pipe(yaml.LookupCreate(entry.value.YNode().Kind, valueKey))
				if err != nil {
					return nil, err
				}
				if len(byValue) > 1 {
					vn.SetYNode(entry.value.YNode())
				} else {
					vn.SetYNode(yaml.NewStringRNode("true").YNode())
				}
				for _, cluster := range clusters {
					chart.ClusterPresets[cluster].Insert(presetKey)
				}
			}
			chart.entries[id] = append(chart.entries[id], entry)
			isOptional := clustersCount < keyCount[path.Dir(valueKey)]
			switch {
			case isOptional && len(byValue) > 1:
				helm.SetOptionalValue(valueKey, entry.value, chart.token)
			case isOptional:
				helm.SetOptional(valueKey, entry.value, chart.token)
			case len(byValue) > 1:
				helm.SetValue(valueKey, entry.value, chart.token)
			}
		}
	}

	resourceBase, err := yaml.Parse(resourceBaseYaml)
	if err != nil {
		panic(err)
	}
	for id, entries := range chart.entries {
		resKey := resourceKey(id)

		rn := resourceBase.Copy()
		rn.SetApiVersion(id.ApiVersion())
		rn.SetName(id.Name)
		rn.SetKind(id.Kind)
		if id.Namespace != "" {
			rn.SetNamespace(id.Namespace)
		}
		export.SetObjectPath(rn, false)
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
		export.SetObjectPath(rn, false)
		if keyCount[resKey] < len(buffers) {
			helm.SetOptional(resKey, rn, chart.token)
			for presetKey := range resPresets[resKey] {
				var values *yaml.RNode
				if _, isCluster := buffers[presetKey]; isCluster {
					values = chart.ClusterValues[presetKey]
				} else {
					var err error
					values, err = presets.Pipe(yaml.LookupCreate(yaml.MappingNode, presetKey))
					if err != nil {
						panic(err)
					}
				}
				err := values.PipeE(
					yaml.LookupCreate(yaml.ScalarNode, resKey),
					yaml.Set(yaml.NewStringRNode("true")),
				)
				if err != nil {
					panic(err)
				}
			}
		}
		rn.YNode().HeadComment = "HELM" + chart.token + `: {{- include "merge_presets" . -}}` + "\n" + rn.YNode().HeadComment
		yutil.FixComments(rn.YNode())
		chart.Templates = append(chart.Templates, rn)
	}

	yutil.SortMapKeys(presets)
	for preset := range allPresets {
		v, err := presets.Pipe(yaml.Lookup(preset))
		if err != nil || v.IsNil() {
			continue
		}
		yutil.SortMapKeys(v)
	}

	return chart, nil
}

func SaveClusterCharts(fSys filesys.FileSystem, dir string, chart *HelmChart) error {
	for cluster, presets := range chart.ClusterPresets {
		kustPath := filepath.Join(dir, cluster, "kustomization.yaml")
		chartHome, err := filepath.Rel(filepath.Dir(kustPath), filepath.Dir(chart.Path))
		clusterChart := *chart.HelmChart

		globalValues, err := chart.ClusterValues[cluster].Map()
		if err != nil {
			panic(err)
		}
		clusterChart.ValuesInline = map[string]any{
			"presets": slices.Sorted(maps.Keys(presets)),
		}
		if len(globalValues) > 0 {
			clusterChart.ValuesInline["global"] = globalValues
		}
		clusterKust := &types.Kustomization{
			HelmGlobals: &types.HelmGlobals{
				ChartHome: chartHome,
			},
			HelmCharts: []types.HelmChart{clusterChart},
		}
		clusterKust.Kind = types.KustomizationKind
		data, err := yaml.Marshal(clusterKust)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(kustPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(kustPath, data, 0o644); err != nil {
			return err
		}
	}

	return nil
}
