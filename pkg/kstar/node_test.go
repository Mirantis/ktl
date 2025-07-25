package kstar

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/Mirantis/ktl/pkg/kquery"
	"github.com/google/go-cmp/cmp"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	//go:embed testdata/pods.yaml
	podsYaml []byte
	pods, _  = (&kio.ByteReader{Reader: bytes.NewBuffer(podsYaml)}).Read()
)

func cloneNodes(rnodes ...*yaml.RNode) *Nodes {
	clones := make([]*yaml.RNode, len(rnodes))
	for idx := range rnodes {
		ynode := yaml.CopyYNode(rnodes[idx].YNode())
		clones[idx] = yaml.NewRNode(ynode)
	}

	return &Nodes{kquery.MakeNodes(clones...)}
}

func TestNode(t *testing.T) {
	tests := []struct {
		name    string
		input   *Nodes
		script  string
		want    starlark.Value
		wantErr bool
	}{
		{
			name:  "scalar-labels",
			input: cloneNodes(pods...),
			script: `
result = [
	nodes[0].metadata.labels.app,
	nodes[1].metadata.labels.app,
]`,
			want: starlark.NewList([]starlark.Value{
				starlark.String("app1"),
				starlark.String("app2"),
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
				test.script,
				starlark.StringDict{
					"nodes": test.input,
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
