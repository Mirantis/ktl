package config

const DefaultFileName = "rekustomization.yaml"

type ClusterGroup struct {
	Group string   `json:"group" yaml:"group"`
	Names []string `json:"names" yaml:"names"`
}

type SkipRule struct {
	MatchResources string `json:"matchResources" yaml:"matchResources"`
	Field          string `json:"field" yaml:"field"`
}

type Rekustomization struct {
	Clusters            []*ClusterGroup `json:"clusters" yaml:"clusters"`
	Namespaces          []string        `json:"namespaces" yaml:"namespaces"`
	NamespacedResources []string        `json:"namespacedResources" yaml:"namespacedResources"`
	ClusterResources    []string        `json:"clusterResources" yaml:"clusterResources"`
	LabelSelectors      []string        `json:"labelSelectors" yaml:"labelSelectors"`
	SkipRules           []*SkipRule     `json:"skip" yaml:"skip"`
}
