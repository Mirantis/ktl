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

type Rekustomization struct {
	Clusters            []*ClusterGroup `json:"clusters" yaml:"clusters"`
	Namespaces          []string        `json:"namespaces" yaml:"namespaces"`
	NamespacedResources []string        `json:"namespacedResources" yaml:"namespacedResources"`
	ClusterResources    []string        `json:"clusterResources" yaml:"clusterResources"`
	LabelSelectors      []string        `json:"labelSelectors" yaml:"labelSelectors"`
	SkipRules           []*SkipRule     `json:"skip" yaml:"skip"`
}
