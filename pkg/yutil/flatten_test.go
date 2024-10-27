package yutil_test

import (
	_ "embed"
	"log/slog"
	"strings"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/yutil"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	//go:embed testdata/deployment.yaml
	testdataDeployment []byte
	//go:embed testdata/deployment-min-indent.yaml
	testdataDeploymentMinIndent []byte
	flattenedDeployment         = map[string]string{
		"/apiVersion":          "apps/v1",
		"/kind":                "Deployment",
		"/metadata/labels/abc": "def",
		"/metadata/labels/app": "test-app",
		"/metadata/labels/xyz": "123",
		"/metadata/name":       "test-deployment",
		"/metadata/namespace":  "test-namespace",
		"/metadata/finalizers": strings.Join([]string{
			`- fin-1`,
			`- fin-2`,
		}, "\n"),
		"/spec/replicas": "1",

		"/spec/selector/matchLabels/app":     "test-app",
		"/spec/template/metadata/labels/app": "test-app",

		"/spec/template/spec/containers/[name=app]/image": "app-image:1.2",
		"/spec/template/spec/containers/[name=app]/name":  "app",

		"/spec/template/spec/containers/[name=app]/volumeMounts/[mountPath=~1tmp]/mountPath": "/tmp",
		"/spec/template/spec/containers/[name=app]/volumeMounts/[mountPath=~1tmp]/name":      "tmp",

		"/spec/template/spec/containers/[name=sidecar]/image": "sidecar-image:3.4",
		"/spec/template/spec/containers/[name=sidecar]/name":  "sidecar",

		"/spec/template/spec/volumes/[name=tmp]/name":     "tmp",
		"/spec/template/spec/volumes/[name=tmp]/emptyDir": "{}",
	}

	//go:embed testdata/custom.yaml
	testdataCustom  []byte
	flattenedCustom = map[string]string{
		"/kind":          "MyCustomResource",
		"/metadata/name": "my-custom-resource",
		"/spec/entries": strings.Join([]string{
			`- name: a`,
			`  attr: 1`,
			`- name: b`,
			`  attr: 2`,
		}, "\n"),
		"/spec/associativeListEntries/[name=c]/attr": "3",
		"/spec/associativeListEntries/[name=c]/name": "c",
		"/spec/associativeListEntries/[name=d]/attr": "4",
		"/spec/associativeListEntries/[name=d]/name": "d",
	}
)

func TestFlatten(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	tests := []struct {
		name  string
		input []byte
		want  map[string]string
	}{
		{
			name:  "deployment",
			input: testdataDeployment,
			want:  flattenedDeployment,
		},
		{
			name:  "deployment in kustomize format",
			input: testdataDeploymentMinIndent,
			want:  flattenedDeployment,
		},
		{
			name:  "custom resource",
			input: testdataCustom,
			want:  flattenedCustom,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := map[string]string{}
			src, _ := kio.FromBytes(test.input)
			for path, value := range yutil.Flatten(src[0]) {
				s, _ := value.String()
				got[path.String()] = strings.TrimRight(s, "\n")
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("-want, +got:\n%v", diff)
			}
		})
	}
}

func TestFlattenRebuild(t *testing.T) {
	tests := []struct {
		name string
		body []byte
	}{
		{name: "deployment", body: testdataDeployment},
		{name: "deployment in kustomize format", body: testdataDeploymentMinIndent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			src, _ := kio.FromBytes(test.body)
			want, _ := src[0].String()
			dst := yaml.NewMapRNode(&map[string]string{})
			for path, value := range yutil.Flatten(src[0]) {
				rn, err := dst.Pipe(yaml.LookupCreate(value.YNode().Kind, path...))
				if err != nil {
					t.Fatal(err)
				}
				rn.SetYNode(value.YNode())
			}
			got, err := dst.String()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("-want, +got:\n%v", diff)
			}
		})
	}
}
