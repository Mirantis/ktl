package yutil

import (
	"fmt"
	"sort"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type contentKV []*yaml.Node

func (kv contentKV) Len() int {
	return len(kv) / 2
}

func (kv contentKV) Less(a, b int) bool {
	return strings.Compare(kv[a*2].Value, kv[b*2].Value) < 0
}

func (kv contentKV) Swap(a, b int) {
	kv[a*2], kv[b*2] = kv[b*2], kv[a*2]
	kv[a*2+1], kv[b*2+1] = kv[b*2+1], kv[a*2+1]
}

func SortMapKeys(rn *yaml.RNode) {
	if rn.YNode().Kind != yaml.MappingNode {
		panic(fmt.Errorf("unable to sort non-mapping type %v", rn.YNode().Kind))
	}
	sort.Sort(contentKV(rn.YNode().Content))
}
