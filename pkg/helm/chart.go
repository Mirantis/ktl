package helm

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/resource"
	"github.com/Mirantis/rekustomize/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//go:embed data/_helpers.tpl
var helpersTpl []byte

type chartValues map[string]*yaml.Node

func (cv chartValues) asMap() map[string]any {
	rn := yaml.NewMapRNode(nil)
	for name, value := range cv {
		if err := rn.SetMapField(yaml.NewRNode(value), name); err != nil {
			panic(err)
		}
	}
	body, err := rn.MarshalJSON()
	if err != nil {
		panic(err)
	}
	result := map[string]any{}
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}
	return result
}

type Chart struct {
	meta      types.HelmChart
	templates map[resid.ResId]*yaml.RNode

	token          string
	presetValues   map[string]chartValues
	inlineValues   map[types.ClusterId]chartValues
	clusterPresets map[types.ClusterId]sets.String
	clusters       *types.ClusterIndex
	clusterIds     []types.ClusterId
}

func NewChart(meta types.HelmChart, clusters *types.ClusterIndex) *Chart {
	chart := &Chart{
		meta:           meta,
		clusters:       clusters,
		clusterIds:     clusters.Ids(),
		token:          rand.String(8),
		presetValues:   map[string]chartValues{},
		inlineValues:   map[types.ClusterId]chartValues{},
		clusterPresets: map[types.ClusterId]sets.String{},
		templates:      map[resid.ResId]*yaml.RNode{},
	}
	return chart
}

func (chart *Chart) templateName(id resid.ResId) string {
	return strings.ToLower(fmt.Sprintf("%s-%s.yaml", id.Name, id.Kind))
}

func (chart *Chart) storeTemplates(fileSys filesys.FileSystem, dir string) error {
	templatePrefix := []byte("# HELM" + chart.token + ": ")
	store := &resource.FileStore{
		Dir:           filepath.Join(dir, "templates"),
		FileSystem:    fileSys,
		NameGenerator: chart.templateName,
		PostProcessor: func(_ string, body []byte) []byte {
			return bytes.ReplaceAll(body, templatePrefix, []byte{})
		},
	}
	if err := store.WriteAll(maps.All(chart.templates)); err != nil {
		return err
	}

	return fileSys.WriteFile(filepath.Join(store.Dir, "_helpers.tpl"), helpersTpl)
}

func (chart *Chart) storeValues(fileSys filesys.FileSystem, dir string) error {
	presets := yaml.NewMapRNode(nil)
	presetNames := slices.Sorted(maps.Keys(chart.presetValues))
	for _, presetName := range presetNames {
		vars := chart.presetValues[presetName]
		preset := yaml.NewMapRNode(nil)
		varNames := slices.Sorted(maps.Keys(vars))
		for _, varName := range varNames {
			value := yaml.NewRNode(vars[varName])
			if err := preset.SetMapField(value, varName); err != nil {
				panic(err)
			}
		}
		if err := presets.SetMapField(preset, presetName); err != nil {
			panic(err)
		}
	}

	root := yaml.NewMapRNode(nil)
	if err := root.SetMapField(yaml.NewMapRNode(nil), "global"); err != nil {
		panic(err)
	}
	if err := root.SetMapField(presets, "preset_values"); err != nil {
		panic(err)
	}
	return fileSys.WriteFile(filepath.Join(dir, "values.yaml"), []byte(root.MustString()))
}

func (chart *Chart) Store(fileSys filesys.FileSystem, dir string) error {
	if err := chart.storeTemplates(fileSys, dir); err != nil {
		return err
	}
	if err := chart.storeValues(fileSys, dir); err != nil {
		return err
	}
	metaBytes, err := yaml.Marshal(chart.meta)
	if err != nil {
		panic(err)
	}
	body := "apiVersion: v2\n" + string(metaBytes)
	return fileSys.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(body))
}

func (chart *Chart) Instance(cluster types.ClusterId) types.HelmChart {
	helmChart := chart.meta
	helmChart.ValuesInline = map[string]any{}
	if presets, found := chart.clusterPresets[cluster]; found {
		helmChart.ValuesInline["presets"] = slices.Sorted(maps.Keys(presets))
	}
	if inline, found := chart.inlineValues[cluster]; found {
		helmChart.ValuesInline["global"] = inline.asMap()
	}
	return helmChart
}

func (chart *Chart) Instances(clusters ...types.ClusterId) []types.HelmChart {
	helmCharts := []types.HelmChart{}
	for _, cluster := range clusters {
		helmCharts = append(helmCharts, chart.Instance(cluster))
	}
	return helmCharts
}

func (chart *Chart) Add(id resid.ResId, resources map[types.ClusterId]*yaml.RNode) error {
	if _, exists := chart.templates[id]; exists {
		return fmt.Errorf("resource already added: %s", id)
	}

	schema := openapi.SchemaForResourceType(id.AsTypeMeta())
	it := resource.NewIterator(resources, schema)
	builder := resource.NewBuilder(id)
	varPrefix := id.Kind + "/" + id.Name
	if id.Namespace != "" {
		varPrefix = id.Namespace + "/" + varPrefix
	}
	occurances := []int{len(chart.clusterIds)}

	for it.Next() {
		path := it.Path()
		depth := len(path)
		occurances = append(occurances, slices.Repeat([]int{0}, max(0, depth+2-len(occurances)))...)
		occurances[depth+1] = len(it.Clusters())
		isOptional := occurances[depth+1] < occurances[depth]
		varName := strings.TrimSuffix(varPrefix+path.String(), "/")
		variants := resource.GroupByValue(it.Values())
		value := chart.value(varName, variants, isOptional)
		if isOptional {
			chart.setOptional(varName, value)
		}
		builder.Set(path, value)
	}

	if err := it.Error(); err != nil {
		return err
	}

	rn := builder.RNode()
	headComment := fmt.Sprintf(`HELM%s: {{- include "merge_presets" . -}}`, chart.token)
	if rn.YNode().HeadComment != "" {
		headComment += "\n" + rn.YNode().HeadComment
	}
	rn.YNode().HeadComment = headComment
	fixComments(rn.YNode())
	chart.templates[id] = rn

	return nil
}

func (chart *Chart) value(variable string, variants []*resource.ValueGroup, optional bool) *yaml.Node {
	if len(variants) < 1 {
		panic(fmt.Errorf("missing values: %s", variable))
	}

	if len(variants) == 1 {
		variant := variants[0]
		value := variant.Value
		if optional {
			preset := chart.values(variant.Clusters)
			preset[variable] = &yaml.Node{Kind: yaml.ScalarNode, Value: "true"}
		}
		return value
	}

	for _, variant := range variants {
		preset := chart.values(variant.Clusters)
		preset[variable] = variant.Value
	}

	node := &yaml.Node{
		Kind:        yaml.ScalarNode,
		LineComment: fmt.Sprintf("HELM%s: {{ index .Values.global \"%s\" }}", chart.token, variable),
	}
	return node
}

func (chart *Chart) values(ids []types.ClusterId) chartValues {
	if len(ids) == 0 {
		panic(fmt.Errorf("missing clusters"))
	}

	preset := chart.clusters.Group(ids...)
	if preset == chart.clusters.Cluster(ids[0]).Name {
		cluster := ids[0]
		values, exists := chart.inlineValues[cluster]
		if !exists {
			values = chartValues{}
			chart.inlineValues[cluster] = values
		}
		return values
	} else {
		values, exists := chart.presetValues[preset]
		if !exists {
			values = chartValues{}
			chart.presetValues[preset] = values
			for _, cluster := range ids {
				clusterPresets, clusterExists := chart.clusterPresets[cluster]
				if !clusterExists {
					clusterPresets = make(sets.String)
					chart.clusterPresets[cluster] = clusterPresets
				}
				clusterPresets.Insert(preset)
			}
		}
		return values
	}
}

func (chart *Chart) setOptional(name string, node *yaml.Node) {
	node.HeadComment = fmt.Sprintf("HELM%s: {{- if index .Values.global \"%s\" }}", chart.token, name)
	node.FootComment = fmt.Sprintf("HELM%s: {{- end }} # %s", chart.token, name)
}
