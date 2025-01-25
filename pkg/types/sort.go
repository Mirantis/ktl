package types

import (
	"sort"
	"strings"
)

var (
	_ sort.Interface = (*orderVariantsByFrequencyAndClusterName)(nil)
	_ sort.Interface = (*orderMValueByDepthAndIndex)(nil)
)

type orderVariantsByFrequencyAndClusterName []map[*Cluster]*NodeMeta

func (o orderVariantsByFrequencyAndClusterName) Len() int      { return len(o) }
func (o orderVariantsByFrequencyAndClusterName) Swap(a, b int) { o[a], o[b] = o[b], o[a] }
func (o orderVariantsByFrequencyAndClusterName) Less(a, b int) bool {
	va, vb := o[a], o[b]
	if byFrequency := len(va) - len(vb); byFrequency != 0 {
		return byFrequency < 0
	}
	nameA := ""
	nameB := ""
	for clusterA := range va {
		if nameA == "" || strings.Compare(nameA, clusterA.Name) > 0 {
			nameA = clusterA.Name
		}
	}
	for clusterB := range vb {
		if nameB == "" || strings.Compare(nameB, clusterB.Name) > 0 {
			nameB = clusterB.Name
		}
	}
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
