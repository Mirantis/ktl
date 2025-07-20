package query

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestBuilderAttr(t *testing.T) {
	const testYAML = `---
a:
  b:
    c: 1
  d:
  - e: e1
  - e: e2
`
	tests := []struct {
		query string
		want  string
	}{
		{query: "a.b.c", want: "1"},
		{query: "a.b", want: "{c: 1}"},
		{query: "a.d.e", want: "[e1, e2]"},
	}

	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			ynode := yaml.MustParse(testYAML).YNode()
			script := fmt.Sprintf("got = resource.%s", test.query)
			input := starlark.StringDict{
				"resource": &Builder{node: ynode},
			}

			thread := &starlark.Thread{
				Name: "test",
				Print: func(thread *starlark.Thread, msg string) {
					t.Log(msg)
				},
			}

			scriptOut, err := starlark.ExecFileOptions(
				&syntax.FileOptions{TopLevelControl: true},
				thread,
				"test",
				script,
				input,
			)
			if err != nil {
				t.Fatal(err)
			}

			got, ok := scriptOut["got"]
			if !ok {
				t.Fatal("missing result")
			}

			gotBuilder, ok := got.(*Builder)
			if !ok {
				t.Fatalf("unexpected result type: %#v", got)
			}

			gotNode, err := gotBuilder.Node()
			if err != nil {
				t.Fatalf("result error: %v", err)
			}

			gotStr, err := yaml.String(gotNode, yaml.Flow)
			if err != nil {
				t.Fatalf("result YAML error: %v", err)
			}

			if diff := cmp.Diff(test.want, strings.TrimSpace(gotStr)); diff != "" {
				t.Fatalf("+got -want:\n%s", diff)
			}
		})
	}
}
