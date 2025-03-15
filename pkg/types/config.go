package types

import "sigs.k8s.io/kustomize/kyaml/kio/filters"

const DefaultFileName = "rekustomization.yaml"

type ClusterSelector struct {
	Names PatternSelector `yaml:"names"`
	Tags  StrList         `yaml:"tags"`
}

type ResourceSelector struct {
	Names          PatternSelector `yaml:"names"`
	Namespaces     PatternSelector `yaml:"namespaces"`
	Resources      PatternSelector `yaml:"apiResources"`
	LabelSelectors []string        `yaml:"labelSelectors"`
}

type Rekustomization struct {
	Clusters  []ClusterSelector  `yaml:"clusters"`
	Resources []ResourceSelector `yaml:"resources"`
	HelmChart HelmChart          `yaml:"helmChart"`

	Filters []filters.KFilter `yaml:"filters"`
}
