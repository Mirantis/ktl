package helm

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
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

var (
	errDuplicateResource = errors.New("duplicate resource")

	//go:embed data/_helpers.tpl
	helpersTpl []byte
)

type chartValues map[string]*yaml.Node

func (cv chartValues) asMap() map[string]any {
	rNode := yaml.NewMapRNode(nil)
	for name, value := range cv {
		if err := rNode.SetMapField(yaml.NewRNode(value), name); err != nil {
			panic(err)
		}
	}

	body, err := rNode.MarshalJSON()
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
	inlineValues   map[types.ClusterID]chartValues
	clusterPresets map[types.ClusterID]sets.String
	clusters       *types.ClusterIndex
	clusterIDs     []types.ClusterID
}

func NewChart(meta types.HelmChart, clusters *types.ClusterIndex) *Chart {
	const tokenLen = 8
	chart := &Chart{
		meta:           meta,
		clusters:       clusters,
		clusterIDs:     clusters.IDs(),
		token:          rand.String(tokenLen),
		presetValues:   map[string]chartValues{},
		inlineValues:   map[types.ClusterID]chartValues{},
		clusterPresets: map[types.ClusterID]sets.String{},
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

	const errMsgTemplates = "unable to store templates"

	err := store.WriteAll(maps.All(chart.templates))
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgTemplates, err)
	}

	err = fileSys.WriteFile(filepath.Join(store.Dir, "_helpers.tpl"), helpersTpl)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsgTemplates, err)
	}

	return nil
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

	err := fileSys.WriteFile(filepath.Join(dir, "values.yaml"), []byte(root.MustString()))
	if err != nil {
		return fmt.Errorf("unable to store values: %w", err)
	}

	return nil
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

	err = fileSys.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(body))
	if err != nil {
		return fmt.Errorf("unable to store chart: %w", err)
	}

	return nil
}

func (chart *Chart) Instance(cluster types.ClusterID) types.HelmChart {
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

func (chart *Chart) Instances(clusters ...types.ClusterID) []types.HelmChart {
	helmCharts := []types.HelmChart{}
	for _, cluster := range clusters {
		helmCharts = append(helmCharts, chart.Instance(cluster))
	}

	return helmCharts
}

func variableName(id resid.ResId, path types.NodePath) string {
	name := fmt.Sprintf("%s/%s/%s.%s", id.Namespace, id.Kind, id.Name, path)
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimSuffix(name, ".")

	return name
}

func (chart *Chart) Add(resID resid.ResId, resources map[types.ClusterID]*yaml.RNode) error {
	if _, exists := chart.templates[resID]; exists {
		return fmt.Errorf("%w: %s", errDuplicateResource, resID)
	}

	schema := openapi.SchemaForResourceType(resID.AsTypeMeta())
	resIterator := resource.NewIterator(resources, schema)
	builder := resource.NewBuilder(resID)
	occurrences := []int{len(chart.clusterIDs)}

	for resIterator.Next() {
		path := resIterator.Path()
		depth := len(path)
		occurrences = append(occurrences, slices.Repeat([]int{0}, max(0, depth+2-len(occurrences)))...)
		occurrences[depth+1] = len(resIterator.Clusters())
		isOptional := occurrences[depth+1] < occurrences[depth]
		varName := variableName(resID, resIterator.Path())
		variants := resource.GroupByValue(resIterator.Values())
		value := chart.value(varName, variants, isOptional)

		if isOptional {
			chart.setOptional(varName, value)
		}

		_, err := builder.Set(path, value)
		if err != nil {
			return fmt.Errorf("chart builder error: %w", err)
		}
	}

	if err := resIterator.Error(); err != nil {
		return fmt.Errorf("chart iterator error: %w", err)
	}

	resNode := builder.RNode()

	headComment := fmt.Sprintf(`HELM%s: {{- include "merge_presets" . -}}`, chart.token)
	if resNode.YNode().HeadComment != "" {
		headComment += "\n" + resNode.YNode().HeadComment
	}

	resNode.YNode().HeadComment = headComment
	fixComments(resNode.YNode())
	chart.templates[resID] = resNode

	return nil
}

func (chart *Chart) value(variable string, variants []*resource.ValueGroup, optional bool) *yaml.Node {
	if len(variants) == 1 {
		variant := variants[0]
		value := variant.Value

		if optional {
			preset := chart.values(variant.Clusters)
			preset[variable] = yaml.NewStringRNode("enabled").YNode()
		}

		return value
	}

	for _, variant := range variants {
		preset := chart.values(variant.Clusters)
		preset[variable] = variant.Value
	}

	node := yaml.NewScalarRNode("").YNode()
	node.LineComment = fmt.Sprintf("HELM%s: {{ index .Values.global \"%s\" }}", chart.token, variable)

	return node
}

func (chart *Chart) values(ids []types.ClusterID) chartValues {
	preset := chart.clusters.Group(ids...)
	if preset == chart.clusters.Cluster(ids[0]).Name {
		cluster := ids[0]
		values, exists := chart.inlineValues[cluster]

		if !exists {
			values = chartValues{}
			chart.inlineValues[cluster] = values
		}

		return values
	}

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

func (chart *Chart) setOptional(name string, node *yaml.Node) {
	node.HeadComment = fmt.Sprintf("HELM%s: {{- if index .Values.global \"%s\" }}", chart.token, name)
	node.FootComment = fmt.Sprintf("HELM%s: {{- end }} # %s", chart.token, name)
}

func fixMappingNode(node *yaml.Node) {
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		fixComments(value)

		if key.HeadComment != "" && value.HeadComment != "" {
			key.HeadComment += "\n"
		}

		if key.FootComment != "" && value.FootComment != "" {
			value.FootComment += "\n"
		}

		key.HeadComment += value.HeadComment
		key.FootComment = value.FootComment + key.FootComment
		value.HeadComment = ""
		value.FootComment = ""
	}
}

func fixSequenceNode(node *yaml.Node) {
	nodes := node.Content
	if len(nodes) < 1 {
		return
	}

	for i := range len(node.Content) - 1 {
		item, nextItem := nodes[i], nodes[i+1]
		fixComments(item)

		if item.FootComment != "" && nextItem.HeadComment != "" {
			item.FootComment += "\n"
		}

		nextItem.HeadComment = item.FootComment + nextItem.HeadComment
		item.FootComment = ""
	}

	lastItem := nodes[len(nodes)-1]
	fixComments(lastItem)

	if node.FootComment != "" && lastItem.FootComment != "" {
		lastItem.FootComment += "\n"
	}

	node.FootComment = lastItem.FootComment + node.FootComment
	lastItem.FootComment = ""
}

func fixComments(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		fixMappingNode(node)
	case yaml.SequenceNode:
		fixSequenceNode(node)
	}
}
