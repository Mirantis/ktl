package types

import (
	"maps"
	"slices"
	"sort"
	"strings"
)

var (
	_ sort.Interface = (*orderVariantsByFrequencyAndClusterName)(nil)
	_ sort.Interface = (*orderMValueByDepthAndIndex)(nil)
)

type orderVariantsByFrequencyAndClusterName struct {
	items    []map[ClusterId]*NodeMeta
	clusters *ClusterIndex
}

func (o orderVariantsByFrequencyAndClusterName) Len() int {
	return len(o.items)
}

func (o orderVariantsByFrequencyAndClusterName) Swap(a, b int) {
	o.items[a], o.items[b] = o.items[b], o.items[a]
}

func (o orderVariantsByFrequencyAndClusterName) Less(a, b int) bool {
	va, vb := o.items[a], o.items[b]
	if byFrequency := len(va) - len(vb); byFrequency != 0 {
		return byFrequency < 0
	}
	idsA := slices.Collect(maps.Keys(va))
	idsB := slices.Collect(maps.Keys(vb))
	nameA := slices.Sorted(o.clusters.Names(idsA...))[0]
	nameB := slices.Sorted(o.clusters.Names(idsB...))[0]
	return strings.Compare(nameA, nameB) < 0
}

type orderMValueByDepthAndIndex []*MValue

func (o orderMValueByDepthAndIndex) Len() int      { return len(o) }
func (o orderMValueByDepthAndIndex) Swap(a, b int) { o[a], o[b] = o[b], o[a] }
func (o orderMValueByDepthAndIndex) Less(a, b int) bool {
	if byDepth := o[a].depth() - o[b].depth(); byDepth != 0 {
		return byDepth < 0
	}
	if byIndex := o[a].avgIndex() - o[b].avgIndex(); byIndex != 0 {
		return byIndex < 0
	}
	return strings.Compare(o[a].Path.String(), o[b].Path.String()) < 0
}
