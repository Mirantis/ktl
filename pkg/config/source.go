package config

import "github.com/Mirantis/rekustomize/pkg/types"

type Source struct {
	Kustomization string                   `yaml:"kustomization"`
	KubeConfig    string                   `yaml:"kubeconfig"`
	Clusters      []types.ClusterSelector  `yaml:"clusters"`
	Resources     []types.ResourceSelector `yaml:"resources"`
}
