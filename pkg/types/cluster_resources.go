package types

import (
	"iter"

	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type ClusterResources struct {
	Clusters  *ClusterIndex
	Resources map[resid.ResId]map[ClusterID]*yaml.RNode
}

func (res *ClusterResources) All() iter.Seq2[resid.ResId, *yaml.RNode] {
	return func(yield func(resid.ResId, *yaml.RNode) bool) {
		for id, byCluster := range res.Resources {
			for _, rnode := range byCluster {
				if !yield(id, rnode) {
					return
				}
			}
		}
	}
}
