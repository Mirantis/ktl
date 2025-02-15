package types

import "sigs.k8s.io/kustomize/kyaml/kio/filters"

const DefaultFileName = "rekustomization.yaml"

type ClusterGroup struct {
	Names PatternSelector `yaml:"names"`
	Tags  StrList         `yaml:"tags"`
}

type ExportRule struct {
	Names          PatternSelector `yaml:"names"`
	Namespaces     PatternSelector `yaml:"namespaces"`
	Resources      PatternSelector `yaml:"apiResources"`
	LabelSelectors []string        `yaml:"labelSelectors"`
}

type Rekustomization struct {
	ClusterGroups []ClusterGroup `yaml:"clusters"`
	ExportRules   []ExportRule   `yaml:"export"`
	HelmChart     HelmChart      `yaml:"helmChart"`

	Filters []filters.KFilter `yaml:"filters"`
}
