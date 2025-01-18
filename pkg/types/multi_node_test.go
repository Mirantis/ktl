package types_test

import (
	"testing"

	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestMNode(t *testing.T) {
	n1 := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
      - name: app
        image: app:v1
        args: [ a, b, c ]
      - name: sidecar1
        image: sidecar1:v1
`)
	n2 := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
      - name: app
        image: app:v1
        args: [ d, e, f ]
      - name: sidecar1
        image: sidecar1:v2
`)
	n3 := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
      - name: app
        image: app:v1
        args: [ d, e, f ]
      - name: sidecar3
        image: sidecar3:v1
`)
	mn := &types.MNode{}
	mn.Add(&types.Cluster{Name: "c1"}, n1)
	mn.Add(&types.Cluster{Name: "c2"}, n2)
	mn.Add(&types.Cluster{Name: "c3"}, n3)

	got := []string{}
	for _, v := range mn.Values() {
		got = append(got, v.String())
	}
	want := []string{
		"/: [c1,c2,c3]=",
		"/apiVersion: [c1,c2,c3]=apps/v1",
		"/kind: [c1,c2,c3]=Deployment",
		"/metadata: [c1,c2,c3]=",
		"/spec: [c1,c2,c3]=",
		"/metadata/name: [c1,c2,c3]=app",
		"/spec/template: [c1,c2,c3]=",
		"/spec/template/spec: [c1,c2,c3]=",
		"/spec/template/spec/containers: [c1,c2,c3]=",
		"/spec/template/spec/containers/[name=app]: [c1,c2,c3]=",
		"/spec/template/spec/containers/[name=sidecar1]: [c1,c2]=",
		"/spec/template/spec/containers/[name=sidecar3]: [c3]=",
		"/spec/template/spec/containers/[name=app]/name: [c1,c2,c3]=app",
		"/spec/template/spec/containers/[name=sidecar1]/name: [c1,c2]=sidecar1",
		"/spec/template/spec/containers/[name=sidecar3]/name: [c3]=sidecar3",
		"/spec/template/spec/containers/[name=app]/image: [c1,c2,c3]=app:v1",
		"/spec/template/spec/containers/[name=sidecar1]/image: [c1]=sidecar1:v1,[c2]=sidecar1:v2",
		"/spec/template/spec/containers/[name=sidecar3]/image: [c3]=sidecar3:v1",
		"/spec/template/spec/containers/[name=app]/args: [c1]=,[c2,c3]=",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("-want +got:\n%v", diff)
	}
}
