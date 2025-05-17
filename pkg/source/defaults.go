package source

import "github.com/Mirantis/ktl/pkg/types"

//nolint:gochecknoglobals
var defaultResourceSelector = types.ResourceSelector{
	LabelSelectors: []string{
		"!kubernetes.io/bootstrapping",
	},
	Resources: types.PatternSelector{
		Exclude: types.Patterns{
			// namespaced
			"*.coordination.k8s.io",
			"*.discovery.k8s.io",
			"*.events.k8s.io",
			"csistoragecapacities.storage.k8s.io",
			"endpoints",
			"events",
			"jobs",
			"limitranges",
			"pods",
			"replicasets.apps",
			// cluster
			"*.admissionregistration.k8s.io",
			"*.apiregistration.k8s.io",
			"*.flowcontrol.apiserver.k8s.io",
			"*.scheduling.k8s.io",
			"componentstatuses",
			"csinodes.storage.k8s.io",
			"nodes",
			"persistentvolumes",
			"volumeattachments.storage.k8s.io",
		},
	},
}

func defaultResources(selectors []types.ResourceSelector) []types.ResourceSelector {
	labelSelectors := defaultResourceSelector.LabelSelectors
	excludeResources := defaultResourceSelector.Resources.Exclude

	for i := range selectors {
		if len(selectors[i].Resources.Include) == 0 {
			selectors[i].Resources.Exclude = append(selectors[i].Resources.Exclude, excludeResources...)
		}

		selectors[i].LabelSelectors = append(selectors[i].LabelSelectors, labelSelectors...)
	}

	return selectors
}
