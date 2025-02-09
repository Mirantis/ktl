package helm_test

import (
	"embed"
	"testing"

	"github.com/Mirantis/rekustomize/examples"
	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/helm"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//go:embed testdata/chart
//go:embed testdata/chart/templates/_helpers.tpl
var chartFs embed.FS

func TestChart(t *testing.T) {
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

	meta := types.HelmChart{
		Name:    "myapp",
		Version: "v0.1",
	}

	chart := helm.NewChart(meta, clusters)
	id := resid.FromRNode(examples.MyAppDeploymentDevA)
	if err := chart.Add(id, resources); err != nil {
		t.Fatal(err)
	}
	gotFs := filesys.MakeFsInMemory()
	if err := chart.Store(gotFs, "/"); err != nil {
		t.Fatal(err)
	}
	got := e2e.ReadFiles(gotFs, "/")
	want := e2e.ReadFsFiles(chartFs, "testdata/chart")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("chart mismatch, +got -want:\n%s", diff)
	}

	gotInstances := chart.Instances(devA, prodA, prodB, testA, testB)
	wantInstances := []types.HelmChart{meta, meta, meta, meta, meta}
	wantInstances[0].ValuesInline = map[string]any{
		"presets": []string{"dev"},
	}
	wantInstances[1].ValuesInline = map[string]any{
		"presets": []string{"prod", "prod_test"},
		"global": map[string]any{
			"myapp/Deployment/myapp/spec/replicas": 3.0,
		},
	}
	wantInstances[2].ValuesInline = map[string]any{
		"presets": []string{"prod", "prod_test"},
		"global": map[string]any{
			"myapp/Deployment/myapp/spec/replicas": 5.0,
		},
	}
	wantInstances[3].ValuesInline = map[string]any{
		"presets": []string{"prod_test", "test"},
	}
	wantInstances[4].ValuesInline = map[string]any{
		"presets": []string{"prod_test", "test"},
	}
	if diff := cmp.Diff(wantInstances, gotInstances); diff != "" {
		t.Errorf("instances mismatch, +got -want:\n%s", diff)
	}
}
