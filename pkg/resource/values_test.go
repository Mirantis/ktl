package resource_test

import (
	"fmt"
	"maps"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/resource"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestGroupByValue(t *testing.T) {
	n0 := yaml.MustParse(`{ a: 1, b: 2, c: 3 }`)
	n1 := yaml.MustParse(`{ a: 1, c: 3, b: 2 }`)
	n2 := yaml.MustParse(`{ a: 2, b: 3 }`)
	n3 := yaml.MustParse(`{ a: 2, b: 3 }`)
	n4 := yaml.MustParse(`{ a: 2, b: 3 }`)
	input := map[types.ClusterId]*yaml.Node{
		0: yaml.CopyYNode(n0.YNode()),
		1: yaml.CopyYNode(n1.YNode()),
		2: yaml.CopyYNode(n2.YNode()),
		3: yaml.CopyYNode(n3.YNode()),
		4: yaml.CopyYNode(n4.YNode()),
	}
	groups := resource.GroupByValue(maps.All(input))
	got := []string{}
	for _, group := range groups {
		got = append(got, fmt.Sprintf(
			"%v: %s",
			group.Clusters,
			yaml.NewRNode(group.Value).MustString(),
		))
	}
	want := []string{
		"[2 3 4]: {a: 2, b: 3}\n",
		"[0]: {a: 1, b: 2, c: 3}\n",
		"[1]: {a: 1, c: 3, b: 2}\n",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("+got -want:\n%s", diff)
	}
}
