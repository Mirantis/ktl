package yutil_test

import (
	"fmt"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/yutil"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var testNode = yaml.MustParse(`
m: v
sequence:
- a: 1
  b: 2
- c: 3
  d: 4
`)

func TestAllNodes(t *testing.T) {
	want := []string{
		"!!map:",
		"!!str:m", "!!str:v",
		"!!str:sequence",
		"!!seq:",
		"!!map:",
		"!!str:a", "!!int:1",
		"!!str:b", "!!int:2",
		"!!map:",
		"!!str:c", "!!int:3",
		"!!str:d", "!!int:4",
	}
	got := []string{}
	for yn := range yutil.AllNodes(testNode.YNode()) {
		got = append(got, fmt.Sprintf("%v:%v", yn.Tag, yn.Value))
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("-want +got:\n%v", diff)
	}
}
