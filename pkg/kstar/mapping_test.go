package kstar

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestMapping(t *testing.T) {
	pod := yaml.MustParse(`
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: app1
  annotations: {}
  name: demo-app1
`).YNode()

	tests := []struct {
		name    string
		expr    string
		want    starlark.Value
		wantErr bool
	}{
		{
			name: "top-level-scalar-field",
			expr: "node.kind",
			want: starlark.String("Pod"),
		},
		{
			name: "nested-scalar-field",
			expr: "node.metadata.labels.app",
			want: starlark.String("app1"),
		},
		{
			name: "missing-field",
			expr: "node.metadata.labels.missing",
			want: starlark.None,
		},
		{
			name: "truth-true",
			expr: "bool(node.metadata.labels)",
			want: starlark.True,
		},
		{
			name: "truth-false",
			expr: "bool(node.metadata.annotations)",
			want: starlark.False,
		},
		{
			name: "dir",
			expr: "dir(node)",
			want: starlark.NewList([]starlark.Value{
				starlark.String("apiVersion"),
				starlark.String("kind"),
				starlark.String("metadata"),
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const resultVar = "result"
			opts := &syntax.FileOptions{
				TopLevelControl: true,
			}

			thread := &starlark.Thread{
				Name: test.name,
				Print: func(_ *starlark.Thread, msg string) {
					t.Logf("starlark output: %s", msg)
				},
			}
			gotAll, err := starlark.ExecFileOptions(
				opts,
				thread,
				test.name,
				fmt.Sprintf("%s = %s", resultVar, test.expr),
				starlark.StringDict{
					"node": &MappingNode{value: yaml.CopyYNode(pod)},
				},
			)

			got := gotAll[resultVar]

			if err != nil && !test.wantErr {
				t.Fatal(err)
			}

			if err == nil && test.wantErr {
				t.Fatal("want error, got none")
			}

			if diff := cmp.Diff(test.want, got, cmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}
