package resource_test

import (
	"testing"

	"github.com/Mirantis/ktl/pkg/resource"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNodePathString(t *testing.T) {
	tests := map[string]struct {
		input resource.Query
		want  string
	}{
		`nil`: {
			nil,
			"",
		},
		`empty`: {
			resource.Query{},
			"",
		},
		`simple`: {
			resource.Query{"a", "b", "c"},
			"a.b.c",
		},
		`escaped`: {
			resource.Query{"a.b", "c"},
			"[a.b].c",
		},
		`filter`: {
			resource.Query{"[a=b]", "c"},
			"[a=b].c",
		},
		`filter-with-dot`: {
			resource.Query{"[a=b.1]", "c"},
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
		Got     resource.Query `yaml:"got"`
		want    resource.Query
		wantErr bool
	}{
		{
			name:  "list",
			input: `got: [ a, b, c ]`,
			want:  resource.Query{"a", "b", "c"},
		},
		{
			name:  "text",
			input: `got: a.b.c`,
			want:  resource.Query{"a", "b", "c"},
		},
		{
			name:  "condition",
			input: `got: a.[b=1].c`,
			want:  resource.Query{"a", "[b=1]", "c"},
		},
		{
			name:  "glob",
			input: `got: a.*.b`,
			want:  resource.Query{"a", "*", "b"},
		},
		{
			name:  "mixed",
			input: `got: a[b=1].*.[c=2]d`,
			want:  resource.Query{"a[b=1]", "*", "[c=2]d"},
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

func TestNodePathNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input resource.Query
		wantP resource.Query
		wantC []string
		wantE bool
	}{
		{
			name:  "no-conditions",
			input: resource.Query{"a", "b", "c"},
			wantP: resource.Query{"a", "b", "c"},
			wantC: []string{"", "", ""},
		},
		{
			name:  "prefix-only",
			input: resource.Query{"[a=1]b", "*", "[c=2]d"},
			wantP: resource.Query{"b", "*", "d"},
			wantC: []string{"[a=1]", "", "[c=2]"},
		},
		{
			name:  "glob",
			input: resource.Query{"a", "[b=1]", "c"},
			wantP: resource.Query{"a", "*", "c"},
			wantC: []string{"", "", "[b=1]"},
		},
		{
			name:  "overflow",
			input: resource.Query{"a", "b[c=1,d=2]"},
			wantP: resource.Query{"a", "b", "c"},
			wantC: []string{"", "", "[c=1,d=2]"},
		},
		{
			name:  "merge",
			input: resource.Query{"a[b=1]", "[c=2]d"},
			wantP: resource.Query{"a", "d"},
			wantC: []string{"", "[b=1,c=2]"},
		},
		{
			name:  "error",
			input: resource.Query{"a[b=1]", "[c=2]", "d"},
			wantE: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotP, gotC, err := test.input.Normalize()

			if err != nil && test.wantE {
				return
			}

			if err != nil {
				t.Fatalf("want no error, got: %v", err)
			}

			if test.wantE {
				t.Fatalf("want error, got none")
			}

			if diff := cmp.Diff(test.wantP, gotP); diff != "" {
				t.Errorf("path -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(test.wantC, gotC); diff != "" {
				t.Errorf("conditions -want +got:\n%s", diff)
			}
		})
	}
}
