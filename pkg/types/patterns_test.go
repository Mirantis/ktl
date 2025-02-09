package types_test

import (
	"slices"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
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
			name:  `csv`,
			input: ` a , b , c `,
			want:  types.Patterns{"a", "b", "c"},
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

func TestPatternSelectorUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    types.PatternSelector
		wantErr bool
	}{
		{
			name:  `include`,
			input: `include: [ a, b, c ]`,
			want:  types.PatternSelector{Include: types.Patterns{"a", "b", "c"}},
		},
		{
			name:  `exclude`,
			input: `exclude: [ a, b, c ]`,
			want:  types.PatternSelector{Exclude: types.Patterns{"a", "b", "c"}},
		},
		{
			name:  `include+exclude`,
			input: `{ include: [ a, b ], exclude: [ c, d ] }`,
			want: types.PatternSelector{
				Include: types.Patterns{"a", "b"},
				Exclude: types.Patterns{"c", "d"},
			},
		},
		{
			name:  `list`,
			input: `[ a, b, c ]`,
			want:  types.PatternSelector{Include: types.Patterns{"a", "b", "c"}},
		},
		{
			name:  `list-with-exclude`,
			input: `[ a, -b, -c, d ]`,
			want: types.PatternSelector{
				Include: types.Patterns{"a", "d"},
				Exclude: types.Patterns{"b", "c"},
			},
		},
		{
			name:  `string`,
			input: `"a , -b , -c , d"`,
			want: types.PatternSelector{
				Include: types.Patterns{"a", "d"},
				Exclude: types.Patterns{"b", "c"},
			},
		},
		{
			name:  `multi-line`,
			input: "- a\n- -b\n- -c\n- d",
			want: types.PatternSelector{
				Include: types.Patterns{"a", "d"},
				Exclude: types.Patterns{"b", "c"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got types.PatternSelector
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
