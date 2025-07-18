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

func (res *ClusterResources) All(cluster *ClusterID) iter.Seq2[resid.ResId, *yaml.RNode] {
	return func(yield func(resid.ResId, *yaml.RNode) bool) {
		for id, byCluster := range res.Resources {
			for resClusterID, rnode := range byCluster {
				// FIXME: inefficient filtering
				if cluster != nil && resClusterID != *cluster {
					continue
				}

				if !yield(id, rnode) {
					return
				}
			}
		}
	}
}
