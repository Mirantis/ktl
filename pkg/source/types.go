package source

import (
	"github.com/Mirantis/ktl/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type State struct {
	Clusters  *types.ClusterIndex
	Resources map[types.ClusterID][]*yaml.RNode
}

type Impl interface {
	Load(env *types.Env) (*State, error)
}
