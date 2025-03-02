package kustomize_test

import (
	"embed"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/kustomize"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	//go:embed testdata/components
	compFs embed.FS

	//go:embed testdata/dev-cluster-a.yaml
	appDevA string
	//go:embed testdata/test-cluster-a.yaml
	appTestA string
	//go:embed testdata/test-cluster-b.yaml
	appTestB string
	//go:embed testdata/prod-cluster-a.yaml
	appProdA string
	//go:embed testdata/prod-cluster-b.yaml
	appProdB string
)

func TestComponents(t *testing.T) {
	clusters := types.NewClusterIndex()
	devA := clusters.Add(types.Cluster{Name: "dev-a", Tags: []string{"dev"}})
	prodA := clusters.Add(types.Cluster{Name: "prod-a", Tags: []string{"prod"}})
	prodB := clusters.Add(types.Cluster{Name: "prod-b", Tags: []string{"prod"}})
	testA := clusters.Add(types.Cluster{Name: "test-a", Tags: []string{"test"}})
	testB := clusters.Add(types.Cluster{Name: "test-b", Tags: []string{"test"}})
	resources := map[types.ClusterID]*yaml.RNode{
		devA:  yaml.MustParse(appDevA),
		prodA: yaml.MustParse(appProdA),
		prodB: yaml.MustParse(appProdB),
		testA: yaml.MustParse(appTestA),
		testB: yaml.MustParse(appTestB),
	}
	id := resid.FromRNode(resources[devA])
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
