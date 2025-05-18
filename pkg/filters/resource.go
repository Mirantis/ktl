package filters

import (
	"fmt"

	"github.com/Mirantis/ktl/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type ResourceMatcher struct {
	Kind      string            `yaml:"kind"`
	Resources []*types.Selector `yaml:"resources"`
	Except    []*types.Selector `yaml:"except"`
}

func (m *ResourceMatcher) Filter(input *yaml.RNode) (*yaml.RNode, error) {
	match, err := matchSelectors(input, m.Resources)
	if err != nil {
		return nil, err
	} else if len(m.Resources) > 0 && !match {
		return nil, nil //nolint:nilnil
	}

	matchExcept, err := matchSelectors(input, m.Except)
	if err != nil {
		return nil, err
	} else if len(m.Except) > 0 && matchExcept {
		return nil, nil //nolint:nilnil
	}

	return input, nil
}

func matchSelectors(resNode *yaml.RNode, selectors []*types.Selector) (bool, error) {
	id := resid.FromRNode(resNode)
	for _, selector := range selectors {
		if !id.IsSelectedBy(selector.ResId) {
			continue
		}

		matchLabels, err := resNode.MatchesLabelSelector(selector.LabelSelector)
		if err != nil {
			return false, fmt.Errorf("invalid label selector: %w", err)
		} else if !matchLabels {
			continue
		}

		matchAnnotations, err := resNode.MatchesAnnotationSelector(selector.AnnotationSelector)
		if err != nil {
			return false, fmt.Errorf("invalid annotation selector: %w", err)
		} else if !matchAnnotations {
			continue
		}

		return true, nil
	}

	return false, nil
}
