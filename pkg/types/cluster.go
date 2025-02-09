package types

import (
	"fmt"
	"iter"
	"maps"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/RoaringBitmap/roaring/v2"
	"sigs.k8s.io/kustomize/kyaml/sets"
)

type Cluster struct {
	Name string
	Tags []string
}

type ClusterId uint32

type ClusterIndex struct {
	items  []Cluster
	ids    []ClusterId
	byName map[string]ClusterId

	cachedGroups map[string]string
	cachedTags   []string
	cachedTagsCB []*roaring.Bitmap
}

func NewClusterIndex() *ClusterIndex {
	return &ClusterIndex{
		byName:       map[string]ClusterId{},
		cachedGroups: map[string]string{},
	}
}

func (idx *ClusterIndex) All() iter.Seq2[ClusterId, Cluster] {
	if len(idx.items) != len(idx.ids) {
		panic(fmt.Errorf("corrupted cluster index"))
	}
	return func(yield func(ClusterId, Cluster) bool) {
		for i := range len(idx.items) {
			if !yield(idx.ids[i], idx.items[i]) {
				return
			}
		}
	}
}

func (idx *ClusterIndex) Id(name string) (ClusterId, error) {
	id, found := idx.byName[name]
	if found {
		return id, nil
	}
	return math.MaxUint32, fmt.Errorf("cluster not found: %s", name)
}

func (idx *ClusterIndex) Add(cluster Cluster) ClusterId {
	tags := sets.String{}
	tags.Insert(cluster.Tags...)
	id, exists := idx.byName[cluster.Name]

	if !exists {
		id = ClusterId(len(idx.items))
		idx.ids = append(idx.ids, id)
		idx.items = append(idx.items, cluster)
		idx.byName[cluster.Name] = id
	} else {
		tags.Insert(idx.items[id].Tags...)
	}

	idx.items[id] = Cluster{
		Name: cluster.Name,
		Tags: slices.Sorted(maps.Keys(tags)),
	}

	if len(idx.cachedGroups) > 0 {
		idx.cachedGroups = map[string]string{}
	}
	idx.cachedTags = nil
	idx.cachedTagsCB = nil

	return id
}

func (idx *ClusterIndex) Ids() []ClusterId {
	return slices.Clone(idx.ids)
}

func (idx *ClusterIndex) Names(ids ...ClusterId) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, id := range ids {
			cluster := idx.Cluster(id)
			if !yield(cluster.Name) {
				return
			}
		}
	}
}

func (idx *ClusterIndex) checkId(id ClusterId) error {
	if int(id) >= len(idx.items) || int(id) < 0 {
		return fmt.Errorf("cluster id out of range: %v", id)
	}
	return nil
}

func (idx *ClusterIndex) Cluster(id ClusterId) Cluster {
	if err := idx.checkId(id); err != nil {
		panic(err)
	}
	return idx.items[id]
}

func (idx *ClusterIndex) groupKey(ids []ClusterId) string {
	sortedIds := slices.Clone(ids)
	slices.Sort(sortedIds)
	return fmt.Sprintf("%v", sortedIds)
}

func (idx *ClusterIndex) rebuildTags() {
	tagMap := map[string]*roaring.Bitmap{}
	for id, cluster := range idx.items {
		for _, tag := range cluster.Tags {
			bits, found := tagMap[tag]
			if !found {
				bits = roaring.NewBitmap()
				tagMap[tag] = bits
			}
			bits.AddInt(id)
		}
	}
	for tag, bitmap := range tagMap {
		idx.cachedTags = append(idx.cachedTags, tag)
		idx.cachedTagsCB = append(idx.cachedTagsCB, bitmap)
	}
	sort.Sort(&orderTagsBySizeAndName{tags: idx.cachedTags, bitmaps: idx.cachedTagsCB})
}

func (idx *ClusterIndex) tags() iter.Seq2[string, *roaring.Bitmap] {
	if len(idx.cachedTags) == 0 {
		idx.rebuildTags()
	}

	return func(yield func(string, *roaring.Bitmap) bool) {
		for i, tag := range idx.cachedTags {
			if !yield(tag, idx.cachedTagsCB[i]) {
				return
			}
		}
	}
}

func (idx *ClusterIndex) Group(ids ...ClusterId) string {
	if len(ids) == 0 {
		return ""
	}
	key := idx.groupKey(ids)
	if group, cached := idx.cachedGroups[key]; cached {
		return group
	}

	bitmap := roaring.NewBitmap()
	for _, id := range ids {
		if err := idx.checkId(id); err != nil {
			panic(err)
		}
		bitmap.Add(uint32(id))
	}

	if len(idx.items) == int(bitmap.GetCardinality()) {
		return "all-clusters"
	}

	remaining := bitmap.Clone()
	parts := []string{}
	for tag, tagCB := range idx.tags() {
		if tagCB.GetCardinality() != tagCB.AndCardinality(bitmap) {
			continue
		}
		if !tagCB.Intersects(remaining) {
			continue
		}
		remaining.AndNot(tagCB)
		parts = append(parts, tag)
	}

	names := []string{}
	for it := remaining.Iterator(); it.HasNext(); {
		id := it.Next()
		names = append(names, idx.items[id].Name)
	}
	slices.Sort(names)
	parts = append(parts, names...)

	group := strings.Join(parts, "+")
	idx.cachedGroups[key] = group
	return group
}
