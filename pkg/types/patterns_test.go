package types_test

import (
	"slices"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestPatternsUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    types.Patterns
		wantErr bool
	}{
		{
			name:  `simple`,
			input: `[ a, b, c ]`,
			want:  types.Patterns{"a", "b", "c"},
		},
		{
			name:  `patterns`,
			input: `[ "[a-z0-9]", "*b*", "c?d" ]`,
			want:  types.Patterns{"[a-z0-9]", "*b*", "c?d"},
		},
		{
			name:    `json-error`,
			input:   `a,b,c`,
			wantErr: true,
		},
		{
			name:    `syntax-error`,
			input:   `[ "[a" ]`,
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got types.Patterns
			err := yaml.Unmarshal([]byte(test.input), &got)
			if test.wantErr && err == nil {
				t.Fatalf("want err, got none")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("want no err, got: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestPatternSelector(t *testing.T) {
	input := []string{"abc123", "abc", "def123", "abc123x", "abcyz", "def"}
	tests := []struct {
		name     string
		selector types.PatternSelector
		want     []string
		wantErr  string
	}{
		{
			name:     "no patterns",
			want:     slices.Clone(input),
			selector: types.PatternSelector{},
		},
		{
			name: "include+exclude",
			want: []string{"abc123", "abc", "def"},
			selector: types.PatternSelector{
				Include: types.Patterns{"abc*", "def"},
				Exclude: types.Patterns{"*x", "*y*"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.selector.Select(input)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("unexpected result, +got -want:\n%v", diff)
			}
		})
	}
}
