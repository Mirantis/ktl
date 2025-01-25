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
		sort.Sort(orderVariantsByFrequencyAndClusterName(mv.cachedVariants))
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
