package types

import (
	"fmt"
	"iter"
	"slices"
)

type Cluster struct {
	Name string
	Tags []string
}

type ClusterId int

type ClusterIndex struct {
	items  []Cluster
	ids    []ClusterId
	byName map[string]ClusterId
}

func (idx *ClusterIndex) Add(cluster Cluster) ClusterId {
	id, exists := idx.byName[cluster.Name]
	if !exists {
		id = ClusterId(len(idx.items))
		idx.ids = append(idx.ids, id)
		idx.items = append(idx.items, cluster)
	}

	return id
}

func (idx *ClusterIndex) Ids() []ClusterId {
	return slices.Clone(idx.ids)
}

func (idx *ClusterIndex) Names(ids ...ClusterId) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, id := range ids {
			cluster, err := idx.Cluster(id)
			if err != nil {
				panic(err)
			}
			if !yield(cluster.Name) {
				return
			}
		}
	}
}

func (idx *ClusterIndex) Cluster(id ClusterId) (Cluster, error) {
	if int(id) >= len(idx.items) || int(id) < 0 {
		return Cluster{}, fmt.Errorf("cluster id out of range: %v", id)
	}
	return idx.items[id], nil
}
