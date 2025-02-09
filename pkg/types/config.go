package types

const DefaultFileName = "rekustomization.yaml"

type ClusterGroup struct {
	Group string   `json:"group" yaml:"group"`
	Names []string `json:"names" yaml:"names"`
}

type SkipRule struct {
	Resources []*Selector `json:"resources" yaml:"resources"`
	Excluding []*Selector `json:"excluding" yaml:"excluding"`
	Fields    []string    `json:"fields" yaml:"fields"`
}

type ExportRule struct {
	Names          PatternSelector `json:"names" yaml:"names"`
	Namespaces     PatternSelector `json:"namespaces" yaml:"namespaces"`
	Resources      PatternSelector `json:"apiResources" yaml:"apiResources"`
	LabelSelectors []string        `json:"labelSelectors" yaml:"labelSelectors"`
}

type Rekustomization struct {
	Clusters    []ClusterGroup `json:"clusters" yaml:"clusters"`
	ExportRules []ExportRule   `json:"export" yaml:"export"`
	SkipRules   []SkipRule     `json:"skip" yaml:"skip"`
	HelmCharts  []HelmChart    `json:"helmCharts" yaml:"helmCharts"`
}
