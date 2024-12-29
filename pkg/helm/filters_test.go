package helm

import (
	_ "embed"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/yutil"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	//go:embed testdata/deployment.yaml
	deployment string

	//go:embed testdata/deployment.tmpl
	template string
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func TestFilters(t *testing.T) {
	token := "TEST"
	rn := yaml.MustParse(deployment)

	must(SetValue(
		"myapp_myapp_deployment_metadata_labels_env",
		rn, token,
		"metadata", "labels", "env",
	))
	must(SetOptionalValue(
		"myapp_myapp_deployment_spec_replicas",
		rn, token,
		"spec", "replicas",
	))
	must(SetValue(
		"myapp_myapp_deployment_spec_template_spec_containers_myapp_image",
		rn, token,
		"spec", "template", "spec", "containers", "[name=myapp]", "image",
	))
	must(SetOptional(
		"myapp_myapp_deployment_spec_template_spec_containers_myapp_args_enabled",
		rn, token,
		"spec", "template", "spec", "containers", "[name=myapp]", "args",
	))
	yutil.FixComments(rn.YNode())

	got, err := String(token, rn)
	must(err)
	if diff := cmp.Diff(template, got); diff != "" {
		t.Fatalf("-want +got:\n%v", diff)
	}
}
