package kstar

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.starlark.net/starlark"
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
			want: &MappingNode{ynode: yaml.MustParse(`{ key: "value" }`).YNode()},
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
		const resultVar = "result"
		runStarlarkTest(t, test.name,
			fmt.Sprintf("%s = %s", resultVar, test.expr),
			StringDict{
				"node": &MappingNode{ynode: yaml.CopyYNode(pod)},
			},
			test.wantPanic, test.wantErr,
			func(t *testing.T, gotAll StringDict) {
				got := gotAll[resultVar]
				if diff := cmp.Diff(test.want, got, cmpOpts...); diff != "" {
					t.Fatalf("-want +got:\n%s", diff)
				}
			},
		)
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
		node := &MappingNode{ynode: yaml.CopyYNode(cm)}
		runStarlarkTest(t, test.name,
			test.script,
			StringDict{
				"node": node,
			},
			test.wantPanic, test.wantErr,
			func(t *testing.T, _ StringDict) {
				got := node.ynode
				want := yaml.MustParse(test.want).YNode()

				if diff := cmp.Diff(want, got, cmpOpts...); diff != "" {
					t.Fatalf("-want +got:\n%s", diff)
				}
			},
		)
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
				&MappingNode{ynode: yaml.MustParse(`{ key: value }`).YNode()},
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
				&MappingNode{ynode: yaml.MustParse(`{ key: value }`).YNode()},
			}),
		},
	}

	for _, test := range tests {
		node := &MappingNode{ynode: yaml.CopyYNode(cm)}
		runStarlarkTest(t, test.name,
			test.script,
			StringDict{
				"node": node,
			},
			test.wantPanic, test.wantErr,
			func(t *testing.T, gotAll StringDict) {
				got := gotAll["result"]

				if diff := cmp.Diff(test.want, got, cmpOpts...); diff != "" {
					t.Fatalf("-want +got:\n%s", diff)
				}
			},
		)
	}
}

func TestSequenceFilter(t *testing.T) {
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
		`  - key: value1`,
		`  - key: value2`,
		`  - key: value3`,
		`  - other: value4`,
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
			script: `result = node.data.scalars(lambda v: "b" != v)`,
			want: strings.Join([]string{
				`- a`,
				`- c`,
			}, "\n"),
		},
		{
			name:   "mapping",
			script: `result = node.data.mappings(lambda v: v.key != "value2")`,
			want: strings.Join([]string{
				`- key: value1`,
				`- key: value3`,
				`- other: value4`,
			}, "\n"),
		},
		{
			name: "update-mapping",
			script: strings.Join([]string{
				`for item in node.data.mappings(lambda v: v.key != "value2"):`,
				`  item.newKey = "newValue"`,
				`result = node`,
			}, "\n"),
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
				`  - key: value1`,
				`    newKey: newValue`,
				`  - key: value2`,
				`  - key: value3`,
				`    newKey: newValue`,
				`  - other: value4`,
				`    newKey: newValue`,
			}, "\n"),
		},
		{
			name:    "not-callable",
			script:  `node.data.scalars(0)`,
			wantErr: true,
		},
		{
			name: "invalid-callable",
			script: strings.Join([]string{
				`def f(a,b,c):`,
				`  pass`,
				`node.data.scalars(f)`,
			}, "\n"),
			wantErr: true,
		},
	}

	for _, test := range tests {
		node := &MappingNode{ynode: yaml.CopyYNode(cm)}
		runStarlarkTest(t, test.name,
			test.script,
			StringDict{
				"node": node,
			},
			test.wantPanic, test.wantErr,
			func(t *testing.T, gotAll StringDict) {
				got, err := FromStarlark(gotAll["result"])
				if err != nil {
					t.Fatal(err)
				}

				want := yaml.MustParse(test.want).YNode()

				if diff := cmp.Diff(want, got, cmpOpts...); diff != "" {
					t.Fatalf("-want +got:\n%s", diff)
				}
			},
		)
	}
}
