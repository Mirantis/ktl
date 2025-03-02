package helm_test

import (
	"embed"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/helm"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	//go:embed testdata/chart
	//go:embed testdata/chart/templates/_helpers.tpl
	chartFs embed.FS

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

func TestChart(t *testing.T) {
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

	meta := types.HelmChart{
		Name:    "myapp",
		Version: "v0.1",
	}

	chart := helm.NewChart(meta, clusters)
	id := resid.FromRNode(resources[devA])

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
			"myapp/Deployment/myapp.spec.replicas": 3.0,
		},
	}
	wantInstances[2].ValuesInline = map[string]any{
		"presets": []string{"prod", "prod_test"},
		"global": map[string]any{
			"myapp/Deployment/myapp.spec.replicas": 5.0,
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
