package types

import (
	"fmt"
	"maps"
	"slices"
	"sort"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type valueKey = string

type MNode struct {
	index        map[valueKey]*MValue
	cachedValues []*MValue
	clusters     *ClusterIndex
}

func NewMNode(clusters *ClusterIndex) *MNode {
	return &MNode{
		index:    map[valueKey]*MValue{},
		clusters: clusters,
	}
}

func (mn *MNode) Add(cluster ClusterId, rn *yaml.RNode) error {
	if rn.IsNilOrEmpty() {
		return fmt.Errorf("unable to add nil/empty RNode")
	}

	mn.cachedValues = nil
	for _, meta := range WalkNode(rn.YNode()) {
		path := meta.Path()
		key := path.String()
		value, found := mn.index[key]
		if !found {
			value = &MValue{Key: key, Path: path, multiNode: mn}
			mn.index[key] = value
		}
		value.Add(cluster, meta)
	}

	return nil
}

func (mn *MNode) Values() []*MValue {
	if mn.cachedValues == nil {
		mn.cachedValues = slices.Collect(maps.Values(mn.index))
		sort.Sort(orderMValueByDepthAndIndex(mn.cachedValues))
	}
	return mn.cachedValues
}
