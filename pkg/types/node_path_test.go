package types_test

import (
	"testing"

	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNodePathString(t *testing.T) {
	tests := map[string]struct {
		input types.NodePath
		want  string
	}{
		`nil`: {
			nil,
			"",
		},
		`empty`: {
			types.NodePath{},
			"",
		},
		`simple`: {
			types.NodePath{"a", "b", "c"},
			"a.b.c",
		},
		`escaped`: {
			types.NodePath{"a.b", "c"},
			"[a.b].c",
		},
		`filter`: {
			types.NodePath{"[a=b]", "c"},
			"[a=b].c",
		},
		`filter-with-dot`: {
			types.NodePath{"[a=b.1]", "c"},
			"[a=b.1].c",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.input.String()
			if got != test.want {
				t.Errorf("mismatch for path %#v: got %#v, want %#v", test.input, got, test.want)
			}
		})
	}
}

func TestNodePathUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		Got     types.NodePath `yaml:"got"`
		want    types.NodePath
		wantErr bool
	}{
		{
			name:  "list",
			input: `got: [ a, b, c ]`,
			want:  types.NodePath{"a", "b", "c"},
		},
		{
			name:  "json-pointer",
			input: `got: /a/b/c`,
			want:  types.NodePath{"a", "b", "c"},
		},
		{
			name:  "json-pointer-escaped",
			input: `got: /a/b~01/c~1d`,
			want:  types.NodePath{"a", "b~1", "c/d"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := yaml.Unmarshal([]byte(test.input), &test)
			if err != nil && test.wantErr {
				return
			}
			if err != nil {
				t.Fatalf("want no-error, got: %v", err)
			}
			if test.wantErr {
				t.Fatalf("want error, got none")
			}
			if diff := cmp.Diff(test.want, test.Got); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}
