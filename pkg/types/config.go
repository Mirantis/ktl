package types

const DefaultFileName = "rekustomization.yaml"

type ClusterGroup struct {
	Names PatternSelector `json:"names" yaml:"names"`
	Tags  StrList         `json:"tags" yaml:"tags"`
}

type ExportRule struct {
	Names          PatternSelector `json:"names" yaml:"names"`
	Namespaces     PatternSelector `json:"namespaces" yaml:"namespaces"`
	Resources      PatternSelector `json:"apiResources" yaml:"apiResources"`
	LabelSelectors []string        `json:"labelSelectors" yaml:"labelSelectors"`
}

type Rekustomization struct {
	ClusterGroups []ClusterGroup `json:"clusters" yaml:"clusters"`
	ExportRules   []ExportRule   `json:"export" yaml:"export"`
	SkipRules     []SkipRule     `json:"skip" yaml:"skip"`
	HelmCharts    []HelmChart    `json:"helmCharts" yaml:"helmCharts"`
}
