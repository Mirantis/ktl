package resource_test

import (
	"testing"

	"github.com/Mirantis/rekustomize/pkg/resource"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestIterator(t *testing.T) {
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
	idx := types.NewClusterIndex()
	c1 := idx.Add(types.Cluster{Name: "c1"})
	c2 := idx.Add(types.Cluster{Name: "c2"})
	c3 := idx.Add(types.Cluster{Name: "c3"})
	schema := openapi.SchemaForResourceType(yaml.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"})
	it := resource.NewIterator(map[types.ClusterId]*yaml.RNode{
		c1: n1,
		c2: n2,
		c3: n3,
	}, schema)

	got := []string{}
	for it.Next() {
		path := it.Path().String()
		got = append(got, path)
	}
	if err := it.Error(); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"/",
		"/apiVersion",
		"/kind",
		"/metadata",
		"/metadata/name",
		"/spec",
		"/spec/template",
		"/spec/template/spec",
		"/spec/template/spec/containers",
		"/spec/template/spec/containers/[name=app]",
		"/spec/template/spec/containers/[name=app]/name",
		"/spec/template/spec/containers/[name=app]/image",
		"/spec/template/spec/containers/[name=app]/args",
		"/spec/template/spec/containers/[name=sidecar3]",
		"/spec/template/spec/containers/[name=sidecar3]/name",
		"/spec/template/spec/containers/[name=sidecar3]/image",
		"/spec/template/spec/containers/[name=sidecar1]",
		"/spec/template/spec/containers/[name=sidecar1]/name",
		"/spec/template/spec/containers/[name=sidecar1]/image",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("-want +got:\n%v", diff)
	}
}
