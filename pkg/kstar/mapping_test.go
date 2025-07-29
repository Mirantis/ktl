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
		wantErr wantErr
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

			if test.wantErr.check(t, err) {
				return
			}

			got := gotAll[resultVar]
			if diff := cmp.Diff(test.want, got, commonCmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestMappingHasSetField(t *testing.T) {
	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
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
		name    string
		script  string
		want    string
		wantErr wantErr
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

			if test.wantErr.check(t, err) {
				return
			}

			got := node.value
			want := yaml.MustParse(test.want).YNode()

			if diff := cmp.Diff(want, got, cmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestMappingHasGet(t *testing.T) {
	pod := yaml.MustParse(strings.Join([]string{
		`apiVersion: v1`,
		`kind: Pod`,
		`metadata:`,
		`  labels:`,
		`    app: app1`,
		`    "quoted: label": label1`,
		`    unquoted:label: label2`,
		`  annotations: {}`,
		`  name: demo-app1`,
	}, "\n")).YNode()

	tests := []struct {
		name      string
		expr      string
		want      starlark.Value
		wantErr   wantErr
		wantPanic wantPanic
	}{
		{
			name: "top-level-scalar-key",
			expr: `node["kind"]`,
			want: starlark.String("Pod"),
		},
		{
			name: "nested-scalar-key",
			expr: `node["metadata"]["labels"]["app"]`,
			want: starlark.String("app1"),
		},
		{
			name: "missing-key",
			expr: `node["metadata"]["labels"]["missing"]`,
			want: starlark.None,
		},
		{
			name: "quoted-key",
			expr: `node["metadata"]["labels"]["quoted: label"]`,
			want: starlark.String("label1"),
		},
		{
			name: "unquoted-key",
			expr: `node["metadata"]["labels"]["unquoted:label"]`,
			want: starlark.String("label2"),
		},
		{
			name:    "invalid-key-int",
			expr:    `node[1]`,
			wantErr: true,
		},
		{
			name:      "mapping-key",
			expr:      `node[node]`,
			wantPanic: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.wantPanic.recover(t)

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

			if test.wantPanic.check(t) {
				return
			}

			if test.wantErr.check(t, err) {
				return
			}

			got := gotAll[resultVar]
			if diff := cmp.Diff(test.want, got, commonCmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestMappingHasSetKey(t *testing.T) {
	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
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
		wantErr   wantErr
		wantPanic wantPanic
	}{
		{
			name:   "set-self",
			script: `node["data"] = node["data"]`,
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
			script: `node["data"]["strField"] = "test-value"`,
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
			script: `node["data"]["intField"] = 1`,
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
			script: `node["data"]["floatField"] = 1.5`,
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
			script: `node["data"]["boolField"] = True`,
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
			script: `node["data"]["nodeField"] = node["data"]`,
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
			script: `node["data"]["nodeField"] = {"a": 1, "b": 2}`,
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
			script: `node["data"]["nodeField"] = ["a", "b"]`,
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
		{
			name:   "set-unquoted-string-key",
			script: `node["data"]["unquoted/string:key"] = "new-value"`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
				`  unquoted/string:key: new-value`,
			}, "\n"),
		},
		{
			name:   "set-quoted-string-key",
			script: `node["data"]["quoted: string.key"] = "new-value"`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  other: value`,
				`  "quoted: string.key": new-value`,
			}, "\n"),
		},
		{
			name:    "set-invalid-key",
			script:  `node[1] = "new-value"`,
			wantErr: true,
		},
		{
			name:      "set-mapping-key",
			script:    `node[node] = "new-value"`,
			wantPanic: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.wantPanic.recover(t)

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

			if test.wantPanic.check(t) {
				return
			}

			if test.wantErr.check(t, err) {
				return
			}

			got := node.value
			want := yaml.MustParse(test.want).YNode()

			if diff := cmp.Diff(want, got, cmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestMappingMerge(t *testing.T) {
	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
		cmpopts.IgnoreFields(yaml.Node{}, "Line", "Style", "Column", "Tag"),
	})
	left := &MappingNode{value: yaml.MustParse(strings.Join([]string{
		`apiVersion: v1`,
		`kind: ConfigMap`,
		`metadata:`,
		`  name: test-cm`,
		`data:`,
		`  a: 1`,
	}, "\n")).YNode()}

	tests := []struct {
		name      string
		right     starlark.Value
		want      string
		wantErr   wantErr
		wantPanic wantPanic
	}{
		{
			name: "replace-field",
			right: &MappingNode{value: yaml.MustParse(strings.Join([]string{
				`kind: NotConfigMap`,
			}, "\n")).YNode()},
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: NotConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  a: 1`,
			}, "\n"),
		},
		{
			name: "replace-nested",
			right: &MappingNode{value: yaml.MustParse(strings.Join([]string{
				`data:`,
				`  a: 2`,
			}, "\n")).YNode()},
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  a: 2`,
			}, "\n"),
		},
		{
			name: "replace-struct",
			right: FromStringDict(None, StringDict{
				"data": FromStringDict(None, StringDict{
					"a": starlark.MakeInt(2),
				}),
			}),
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  a: 2`,
			}, "\n"),
		},
		{
			name: "append-nested",
			right: &MappingNode{value: yaml.MustParse(strings.Join([]string{
				`data:`,
				`  b: 3`,
			}, "\n")).YNode()},
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  a: 1`,
				`  b: 3`,
			}, "\n"),
		},
		{
			name: "append-struct",
			right: FromStringDict(None, StringDict{
				"data": FromStringDict(None, StringDict{
					"b": starlark.MakeInt(4),
				}),
			}),
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: test-cm`,
				`data:`,
				`  a: 1`,
				`  b: 4`,
			}, "\n"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.wantPanic.recover(t)

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
				"result = (left + right)",
				StringDict{
					"left":  left,
					"right": test.right,
				},
			)

			if err != nil {
				t.Fatalf("script error: %v", err)
			}

			gotExpr, ok := gotAll["result"].(*nodeExpr)
			if !ok {
				t.Fatal("result is not expr")
			}

			gotNode, err := gotExpr.materialize()
			if test.wantErr.check(t, err) {
				return
			}

			if test.wantPanic.check(t) {
				return
			}

			got := gotNode.(*MappingNode).value
			want := yaml.MustParse(test.want).YNode()

			if diff := cmp.Diff(want, got, cmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}
