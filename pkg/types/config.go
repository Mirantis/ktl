package types

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
