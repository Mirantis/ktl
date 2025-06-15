package types

import "github.com/Mirantis/ktl/pkg/apis"

type ClusterSelector struct {
	Names PatternSelector `yaml:"names"`
	Tags  StrList         `yaml:"tags"`
}

func NewClusterSelector(spec *apis.ClusterSelector) (ClusterSelector, error) {
	cs := ClusterSelector{}

	ps, err := NewPatternSelector(spec.GetMatchNames())
	if err != nil {
		return cs, nil
	}

	cs.Tags = StrList{spec.GetAlias()}
	cs.Names = ps

	return cs, nil
}

type ResourceSelector struct {
	Names          PatternSelector `yaml:"names"`
	Namespaces     PatternSelector `yaml:"namespaces"`
	Resources      PatternSelector `yaml:"apiResources"`
	LabelSelectors []string        `yaml:"labelSelectors"`
}

func NewResourceSelector(spec*apis.ResourceMatcher) (ResourceSelector, error) {
	var err error

	names, err := NewPatternSelector(spec.GetMatchNames())
	if err != nil {
		return ResourceSelector{}, nil
	}

	ns, err := NewPatternSelector(spec.GetMatchNamespaces())
	if err != nil {
		return ResourceSelector{}, nil
	}

	res, err := NewPatternSelector(spec.GetMatchApiResources())
	if err != nil {
		return ResourceSelector{}, nil
	}

	return ResourceSelector{
		Names: names,
		Namespaces: ns,
		Resources: res,
		LabelSelectors: spec.GetLabelSelectors(),
	}, nil
}
