package types

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
)

type MValue struct {
	Key  string
	Path NodePath

	variants       map[string]map[*Cluster]*NodeMeta
	cachedVariants []map[*Cluster]*NodeMeta
	cachedIndex    int
}

func (mv *MValue) Add(cluster *Cluster, meta *NodeMeta) {
	mv.cachedVariants = nil
	mv.cachedIndex = -1
	if mv.variants == nil {
		mv.variants = make(map[string]map[*Cluster]*NodeMeta)
	}
	valueHash := meta.Hash()
	byCluster := mv.variants[valueHash]
	if byCluster == nil {
		byCluster = make(map[*Cluster]*NodeMeta)
		mv.variants[valueHash] = byCluster
	}
	byCluster[cluster] = meta
}

func (mv *MValue) Variants() []map[*Cluster]*NodeMeta {
	if mv.cachedVariants == nil {
		mv.cachedVariants = slices.Collect(maps.Values(mv.variants))
		slices.SortFunc(mv.cachedVariants, func(a, b map[*Cluster]*NodeMeta) int {
			if byLen := len(a) - len(b); byLen != 0 {
				return byLen
			}
			minAName := ""
			minBName := ""
			for ca := range a {
				if minAName == "" || strings.Compare(minAName, ca.Name) > 0 {
					minAName = ca.Name
				}
			}
			for cb := range b {
				if minBName == "" || strings.Compare(minBName, cb.Name) > 0 {
					minBName = cb.Name
				}
			}
			return strings.Compare(minAName, minBName)
		})
	}
	return mv.cachedVariants
}

func (mv *MValue) String() string {
	values := []string{}
	for _, variant := range mv.Variants() {
		clusters := slices.Collect(maps.Keys(variant))
		names := []string{}
		for _, cluster := range clusters {
			names = append(names, cluster.Name)
		}
		sort.Strings(names)
		value := fmt.Sprintf("[%s]=%s", strings.Join(names, ","), variant[clusters[0]].Node.Value)
		values = append(values, value)
	}
	return fmt.Sprintf("%s: %s", mv.Key, strings.Join(values, ","))
}

func (mv *MValue) depth() int {
	return len(mv.Path)
}

func (mv *MValue) avgIndex() int {
	if mv.cachedIndex < 0 {
		count := 0
		total := 0
		for _, variant := range mv.Variants() {
			for _, meta := range variant {
				total += meta.Index
				count++
			}
		}
		mv.cachedIndex = total / count
	}
	return mv.cachedIndex
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
