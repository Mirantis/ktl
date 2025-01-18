package types_test

import (
	"fmt"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var testNode = yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: c1
        args: [ a, b ]
      - name: c2
`)

func TestAllNodes(t *testing.T) {
	want := []string{
		"/: !!map:",
		"/apiVersion: !!str:apps/v1",
		"/kind: !!str:Deployment",
		"/spec: !!map:",
		"/spec/template: !!map:",
		"/spec/template/spec: !!map:",
		"/spec/template/spec/containers: !!seq:",
		"/spec/template/spec/containers/[name=c1]: !!map:",
		"/spec/template/spec/containers/[name=c1]/name: !!str:c1",
		"/spec/template/spec/containers/[name=c1]/args: !!seq:",
		"/spec/template/spec/containers/[name=c2]: !!map:",
		"/spec/template/spec/containers/[name=c2]/name: !!str:c2",
	}
	got := []string{}
	for yn, meta := range types.AllNodes(testNode.YNode()) {
		got = append(got, fmt.Sprintf("%v: %v:%v", meta.Path(), yn.Tag, yn.Value))
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("-want +got:\n%v", diff)
	}
}
