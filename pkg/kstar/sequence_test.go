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

func TestSequenceHasGet(t *testing.T) {
	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
		cmpopts.IgnoreFields(yaml.Node{}, "Line", "Style", "Column", "Tag"),
	})
	pod := yaml.MustParse(strings.Join([]string{
		`apiVersion: v1`,
		`kind: ConfigMap`,
		`metadata:`,
		`  name: demo-app1`,
		`data:`,
		`  scalars:`,
		`  - a`,
		`  - b`,
		`  - c`,
		`  mappings:`,
		`  - key: value`,
	}, "\n")).YNode()

	tests := []struct {
		name      string
		expr      string
		want      starlark.Value
		wantErr   wantErr
		wantPanic wantPanic
	}{
		{
			name: "scalar",
			expr: `node.data.scalars[0]`,
			want: starlark.String("a"),
		},
		{
			name: "scalar-tail",
			expr: `node.data.scalars[-2]`,
			want: starlark.String("b"),
		},
		{
			name: "mapping",
			expr: `node.data.mappings[0]`,
			want: &MappingNode{value: yaml.MustParse(`{ key: "value" }`).YNode()},
		},
		{
			name:    "out-of-bounds",
			expr:    `node.data.scalars[200]`,
			wantErr: true,
		},
		{
			name:    "out-of-bounds-tail",
			expr:    `node.data.scalars[-200]`,
			wantErr: true,
		},
		{
			name:    "invalid-key",
			expr:    `node.data.mappings[{}]`,
			wantErr: true,
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
			if diff := cmp.Diff(test.want, got, cmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestSequenceHasSetKey(t *testing.T) {
	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
		cmpopts.IgnoreFields(yaml.Node{}, "Line", "Style", "Column", "Tag"),
	})
	cm := yaml.MustParse(strings.Join([]string{
		`apiVersion: v1`,
		`kind: ConfigMap`,
		`metadata:`,
		`  name: demo-app1`,
		`data:`,
		`  scalars:`,
		`  - a`,
		`  - b`,
		`  - c`,
		`  mappings:`,
		`  - key: value`,
	}, "\n")).YNode()

	tests := []struct {
		name      string
		script    string
		want      string
		wantErr   wantErr
		wantPanic wantPanic
	}{
		{
			name:   "scalar",
			script: `node.data.scalars[0] = "x"`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: demo-app1`,
				`data:`,
				`  scalars:`,
				`  - x`,
				`  - b`,
				`  - c`,
				`  mappings:`,
				`  - key: value`,
			}, "\n"),
		},
		{
			name:   "scalar-tail",
			script: `node.data.scalars[-1] = "z"`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: demo-app1`,
				`data:`,
				`  scalars:`,
				`  - a`,
				`  - b`,
				`  - z`,
				`  mappings:`,
				`  - key: value`,
			}, "\n"),
		},
		{
			name:   "mapping",
			script: `node.data.mappings[0] = { "other": 5 }`,
			want: strings.Join([]string{
				`apiVersion: v1`,
				`kind: ConfigMap`,
				`metadata:`,
				`  name: demo-app1`,
				`data:`,
				`  scalars:`,
				`  - a`,
				`  - b`,
				`  - c`,
				`  mappings:`,
				`  - other: 5`,
			}, "\n"),
		},
		{
			name:    "out-of-bounds",
			script:  `node.data.mappings[200] = "xyz"`,
			wantErr: true,
		},
		{
			name:    "out-of-bounds-tail",
			script:  `node.data.mappings[-200] = "xyz"`,
			wantErr: true,
		},
		{
			name:    "invalid-key",
			script:  `node.data.mappings[{}] = "xyz"`,
			wantErr: true,
		},
		{
			name:    "invalid-value",
			script:  `node.data.mappings[0] = lambda _:_`,
			wantErr: true,
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

func TestSequenceIter(t *testing.T) {
	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
		cmpopts.IgnoreFields(yaml.Node{}, "Line", "Style", "Column", "Tag"),
	})
	cm := yaml.MustParse(strings.Join([]string{
		`apiVersion: v1`,
		`kind: ConfigMap`,
		`metadata:`,
		`  name: demo-app1`,
		`data:`,
		`  scalars:`,
		`  - a`,
		`  - b`,
		`  - c`,
		`  mappings:`,
		`  - key: value`,
	}, "\n")).YNode()

	tests := []struct {
		name      string
		script    string
		want      starlark.Value
		wantErr   wantErr
		wantPanic wantPanic
	}{
		{
			name:   "scalar-list",
			script: `result = list(node.data.scalars)`,
			want: starlark.NewList([]starlark.Value{
				starlark.String("a"),
				starlark.String("b"),
				starlark.String("c"),
			}),
		},
		{
			name:   "mapping-list",
			script: `result = list(node.data.mappings)`,
			want: starlark.NewList([]starlark.Value{
				&MappingNode{value: yaml.MustParse(`{ key: value }`).YNode()},
			}),
		},
		{
			name: "scalar-loop",
			script: strings.Join([]string{
				`result = []`,
				`for item in node.data.scalars:`,
				`  result.append(item)`,
			}, "\n"),
			want: starlark.NewList([]starlark.Value{
				starlark.String("a"),
				starlark.String("b"),
				starlark.String("c"),
			}),
		},
		{
			name: "mapping-loop",
			script: strings.Join([]string{
				`result = []`,
				`for item in node.data.mappings:`,
				`  result.append(item)`,
			}, "\n"),
			want: starlark.NewList([]starlark.Value{
				&MappingNode{value: yaml.MustParse(`{ key: value }`).YNode()},
			}),
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
			gotAll, err := starlark.ExecFileOptions(
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

			got := gotAll["result"]

			if diff := cmp.Diff(test.want, got, cmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}
