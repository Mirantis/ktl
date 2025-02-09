package kustomize_test

import (
	"embed"
	"testing"

	"github.com/Mirantis/rekustomize/examples"
	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/kustomize"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//go:embed testdata/components
var compFs embed.FS

func TestComponents(t *testing.T) {
	clusters := types.NewClusterIndex()
	devA := clusters.Add(types.Cluster{Name: "dev-a", Tags: []string{"dev"}})
	prodA := clusters.Add(types.Cluster{Name: "prod-a", Tags: []string{"prod"}})
	prodB := clusters.Add(types.Cluster{Name: "prod-b", Tags: []string{"prod"}})
	testA := clusters.Add(types.Cluster{Name: "test-a", Tags: []string{"test"}})
	testB := clusters.Add(types.Cluster{Name: "test-b", Tags: []string{"test"}})
	resources := map[types.ClusterId]*yaml.RNode{
		devA:  examples.MyAppDeploymentDevA,
		prodA: examples.MyAppDeploymentProdA,
		prodB: examples.MyAppDeploymentProdB,
		testA: examples.MyAppDeploymentTestA,
		testB: examples.MyAppDeploymentTestB,
	}
	id := resid.FromRNode(examples.MyAppDeploymentDevA)
	comps := kustomize.NewComponents(clusters)
	if err := comps.Add(id, resources); err != nil {
		t.Fatal(err)
	}
	gotFs := filesys.MakeFsInMemory()
	if err := comps.Store(gotFs, "/"); err != nil {
		t.Fatal(err)
	}
	got := e2e.ReadFiles(gotFs, "/")
	want := e2e.ReadFsFiles(compFs, "testdata/components")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("chart mismatch, +got -want:\n%s", diff)
	}
}
