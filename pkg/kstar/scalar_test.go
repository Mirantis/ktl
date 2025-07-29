package kstar

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.starlark.net/starlark"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestScalarValue(t *testing.T) {
	tests := []struct {
		name      string
		input     *ScalarNode
		want      starlark.Value
		wantErr   wantErr
		wantPanic wantPanic
	}{
		{
			name:  "string",
			input: &ScalarNode{value: yaml.NewStringRNode("test-value").YNode()},
			want:  starlark.String("test-value"),
		},
		{
			name:  "int",
			input: &ScalarNode{value: yaml.NewScalarRNode("1").YNode()},
			want:  starlark.MakeInt64(1),
		},
		{
			name:  "float",
			input: &ScalarNode{value: yaml.NewScalarRNode("1.5").YNode()},
			want:  starlark.Float(1.5),
		},
		{
			name:  "true",
			input: &ScalarNode{value: yaml.NewScalarRNode("true").YNode()},
			want:  starlark.True,
		},
		{
			name:  "false",
			input: &ScalarNode{value: yaml.NewScalarRNode("false").YNode()},
			want:  starlark.False,
		},
		{
			name: "invalid-nan",
			input: &ScalarNode{value: &yaml.Node{
				Tag:   "!!int",
				Value: "not-a-number",
			}},
			wantErr: true,
		},
		{
			name:      "panic-map",
			input:     &ScalarNode{value: yaml.NewMapRNode(nil).YNode()},
			wantPanic: true,
		},
		{
			name:      "panic-list",
			input:     &ScalarNode{value: yaml.NewListRNode().YNode()},
			wantPanic: true,
		},
		{
			name:      "panic-null",
			input:     &ScalarNode{value: yaml.MakeNullNode().YNode()},
			wantPanic: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.wantPanic.recover(t)

			got, err := test.input.Value()

			if test.wantPanic.check(t) {
				return
			}

			if test.wantErr.check(t, err) {
				return
			}

			if diff := cmp.Diff(test.want, got, cmpOpts...); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}
