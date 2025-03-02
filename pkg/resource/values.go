package resource

import (
	"crypto/sha256"
	"fmt"
	"iter"
	"maps"
	"slices"
	"sort"

	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type ValueGroup struct {
	Value    *yaml.Node
	Clusters []types.ClusterID
	values   []*yaml.Node
}

func GroupByValue(values iter.Seq2[types.ClusterID, *yaml.Node]) []*ValueGroup {
	groups := map[string]*ValueGroup{}

	for cluster, node := range values {
		// TODO: use Encoder/bytes to avoid bytes->string->bytes
		data, err := yaml.String(node, yaml.Flow)
		if err != nil {
			panic(fmt.Errorf("corrupted yaml"))
		}
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
		group, exists := groups[hash]
		if !exists {
			group = &ValueGroup{
				Value: yaml.CopyYNode(node),
			}
			groups[hash] = group
		}
		group.values = append(group.values, node)
		group.Clusters = append(group.Clusters, cluster)
	}

	result := slices.Collect(maps.Values(groups))
	for _, group := range result {
		sort.Sort((*valueGroupClustersOrder)(group))
	}
	sort.Sort(valueGroupOrder(result))
	return result
}

type valueGroupOrder []*ValueGroup

func (o valueGroupOrder) Len() int      { return len(o) }
func (o valueGroupOrder) Swap(a, b int) { o[a], o[b] = o[b], o[a] }
func (o valueGroupOrder) Less(a, b int) bool {
	va, vb := o[a], o[b]
	if d := len(va.Clusters) - len(vb.Clusters); d != 0 {
		return d > 0 // descending
	}
	return va.Clusters[0] < vb.Clusters[0]
}

type valueGroupClustersOrder ValueGroup

func (o *valueGroupClustersOrder) Len() int { return len(o.Clusters) }
func (o *valueGroupClustersOrder) Swap(a, b int) {
	o.Clusters[a], o.Clusters[b] = o.Clusters[b], o.Clusters[a]
	o.values[a], o.values[b] = o.values[b], o.values[a]
}
func (o *valueGroupClustersOrder) Less(a, b int) bool {
	return o.Clusters[a] < o.Clusters[b]
}
