package types

import (
	"errors"
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

var (
	errIndexInvalid   = errors.New("invalid cluster index")
	errIndexNotFound  = errors.New("cluster not found")
	errIndexInvalidID = errors.New("invalid cluster ID")
)

type Cluster struct {
	Name string
	Tags []string
}

type ClusterID uint32

type ClusterIndex struct {
	items  []Cluster
	ids    []ClusterID
	byName map[string]ClusterID

	cachedGroups map[string]string
	cachedTags   []string
	cachedTagsCB []*roaring.Bitmap
}

func NewClusterIndex() *ClusterIndex {
	return &ClusterIndex{
		byName:       map[string]ClusterID{},
		cachedGroups: map[string]string{},
	}
}

func BuildClusterIndex(names []string, groups []ClusterSelector) *ClusterIndex {
	index := NewClusterIndex()
	clusterTags := map[string]sets.String{}

	for _, group := range groups {
		for _, name := range group.Names.Select(names) {
			tags, exists := clusterTags[name]
			if !exists {
				tags = sets.String{}
				clusterTags[name] = tags
			}

			tags.Insert(group.Tags...)
		}
	}

	sortedNames := slices.Sorted(maps.Keys(clusterTags))

	for _, name := range sortedNames {
		index.Add(Cluster{
			Name: name,
			Tags: slices.Sorted(maps.Keys(clusterTags[name])),
		})
	}

	return index
}

func (idx *ClusterIndex) All() iter.Seq2[ClusterID, Cluster] {
	if len(idx.items) != len(idx.ids) {
		panic(errIndexInvalid)
	}

	return func(yield func(ClusterID, Cluster) bool) {
		for i := range len(idx.items) {
			if !yield(idx.ids[i], idx.items[i]) {
				return
			}
		}
	}
}

func (idx *ClusterIndex) ID(name string) (ClusterID, error) {
	id, found := idx.byName[name]
	if found {
		return id, nil
	}

	return math.MaxInt32, fmt.Errorf("%w: %s", errIndexNotFound, name)
}

func (idx *ClusterIndex) Add(cluster Cluster) ClusterID {
	tags := sets.String{}
	tags.Insert(cluster.Tags...)
	clusterID, exists := idx.byName[cluster.Name]

	if !exists {
		clusterID = ClusterID(len(idx.items)) //nolint
		idx.ids = append(idx.ids, clusterID)
		idx.items = append(idx.items, cluster)
		idx.byName[cluster.Name] = clusterID
	} else {
		tags.Insert(idx.items[clusterID].Tags...)
	}

	idx.items[clusterID] = Cluster{
		Name: cluster.Name,
		Tags: slices.Sorted(maps.Keys(tags)),
	}

	if len(idx.cachedGroups) > 0 {
		idx.cachedGroups = map[string]string{}
	}

	idx.cachedTags = nil
	idx.cachedTagsCB = nil

	return clusterID
}

func (idx *ClusterIndex) IDs() []ClusterID {
	return slices.Clone(idx.ids)
}

func (idx *ClusterIndex) Names(ids ...ClusterID) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, id := range ids {
			cluster := idx.Cluster(id)
			if !yield(cluster.Name) {
				return
			}
		}
	}
}

func (idx *ClusterIndex) checkID(id ClusterID) error {
	if int(id) >= len(idx.items) || int(id) < 0 {
		return fmt.Errorf("%w: %v", errIndexInvalidID, id)
	}

	return nil
}

func (idx *ClusterIndex) Cluster(id ClusterID) Cluster {
	if err := idx.checkID(id); err != nil {
		panic(err)
	}

	return idx.items[id]
}

func (idx *ClusterIndex) groupKey(ids []ClusterID) string {
	sortedIDs := slices.Clone(ids)
	slices.Sort(sortedIDs)

	return fmt.Sprintf("%v", sortedIDs)
}

func (idx *ClusterIndex) rebuildTags() {
	tagMap := map[string]*roaring.Bitmap{}

	for clusterID, cluster := range idx.items {
		for _, tag := range cluster.Tags {
			bits, found := tagMap[tag]
			if !found {
				bits = roaring.NewBitmap()
				tagMap[tag] = bits
			}

			bits.AddInt(clusterID)
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

func (idx *ClusterIndex) Group(ids ...ClusterID) string {
	if len(ids) == 0 {
		return ""
	}

	key := idx.groupKey(ids)

	if group, cached := idx.cachedGroups[key]; cached {
		return group
	}

	bitmap := roaring.NewBitmap()

	for _, id := range ids {
		if err := idx.checkID(id); err != nil {
			panic(err)
		}

		bitmap.Add(uint32(id))
	}

	if len(idx.items) == int(bitmap.GetCardinality()) { //nolint
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

	group := strings.Join(parts, "_")
	idx.cachedGroups[key] = group

	return group
}

type orderTagsBySizeAndName struct {
	tags    []string
	bitmaps []*roaring.Bitmap
}

func (o *orderTagsBySizeAndName) Len() int {
	return len(o.tags)
}

func (o *orderTagsBySizeAndName) Swap(a, b int) {
	o.tags[a], o.tags[b] = o.tags[b], o.tags[a]
	o.bitmaps[a], o.bitmaps[b] = o.bitmaps[b], o.bitmaps[a]
}

func (o *orderTagsBySizeAndName) Less(a, b int) bool { //nolint:varnamelen
	delta := o.bitmaps[a].GetCardinality() - o.bitmaps[b].GetCardinality()
	if delta != 0 {
		return delta > 0
	}

	return strings.Compare(o.tags[a], o.tags[b]) < 0
}
