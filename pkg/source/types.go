package source

import (
	"github.com/Mirantis/ktl/pkg/apis"
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

func New(spec *apis.Source) (Impl, error) {
	if implSpec := spec.GetKubeconfig(); implSpec != nil {
		return newKubeconfig(implSpec)
	}

	if implSpec := spec.GetKustomize(); implSpec != nil {
		return newKustomize(implSpec)
	}

	return newKubeconfig(nil)
}

