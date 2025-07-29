package kstar

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestMappingHasAttrs(t *testing.T) {
	pod := yaml.MustParse(strings.Join([]string{
		`apiVersion: v1`,
		`kind: Pod`,
		`metadata:`,
		`  labels:`,
		`    app: app1`,
		`  annotations: {}`,
		`  name: demo-app1`,
	}, "\n")).YNode()

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

func TestMappingHasSetField(t *testing.T) {
	cmpOpts := slices.Concat(cmpOpts, cmp.Options{
		cmpopts.IgnoreFields(yaml.Node{}, "Line", "Style", "Column"),
	})
	cm := yaml.MustParse(strings.Join([]string{
		`apiVersion: v1`,
		`kind: ConfigMap`,
		`metadata:`,
		`  name: test-cm`,
		`data:`,
		`  other: value`,
	}, "\n")).YNode()

	tests := []struct {
		name      string
		script    string
		want      string
		wantErr   bool
	}{
		{
			name:   "set-self",
			script: `node.data = node.data`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
			}, "\n"),
		},
		{
			name:   "set-scalar-string",
			script: `node.data.strField = "test-value"`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
				`  strField: "test-value"`,
			}, "\n"),
		},
		{
			name:   "set-scalar-int",
			script: `node.data.intField = 1`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
				`  intField: 1`,
			}, "\n"),
		},
		{
			name:   "set-scalar-int",
			script: `node.data.floatField = 1.5`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
				`  floatField: 1.5`,
			}, "\n"),
		},
		{
			name:   "set-scalar-bool",
			script: `node.data.boolField = True`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
				`  boolField: true`,
			}, "\n"),
		},
		{
			name:   "set-mapping-node",
			script: `node.data.nodeField = node.data`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
				`  nodeField:`,
				`    other: value`,
			}, "\n"),
		},
		{
			name:   "set-dict",
			script: `node.data.nodeField = {"a": 1, "b": 2}`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
				`  nodeField:`,
				`    a: 1`,
				`    b: 2`,
			}, "\n"),
		},
		{
			name:   "set-list",
			script: `node.data.nodeField = ["a", "b"]`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
				`  nodeField:`,
				`  - a`,
				`  - b`,
			}, "\n"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node := &MappingNode{value: yaml.CopyYNode(cm)}
			opts := &syntax.FileOptions{
				TopLevelControl: true,
			}

			thread := &starlark.Thread{
				Name: test.name,
				Print: func(_ *starlark.Thread, msg string) {
					t.Logf("starlark output: %s", msg)
				},
			}
			_, err := starlark.ExecFileOptions(
				opts,
				thread,
				test.name,
				test.script,
				starlark.StringDict{
					"node": node,
				},
			)

			if err != nil && !test.wantErr {
				t.Fatal(err)
			}

			if err == nil && test.wantErr {
				t.Fatal("want error, got none")
			}

			got := node.value
			want := yaml.MustParse(test.want).YNode()

			if diff := cmp.Diff(want, got, cmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}
