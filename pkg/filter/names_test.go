package filter_test

import (
	"slices"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/filter"
	"github.com/google/go-cmp/cmp"
)

func TestNames(t *testing.T) {
	input := []string{"abc123", "abc", "def123", "abc123x", "abcyz", "def"}
	tests := []struct {
		name     string
		patterns []string
		want     []string
		wantErr  string
	}{
		{
			name:     "no patterns",
			patterns: []string{},
			want:     slices.Clone(input),
		},
		{
			name:     "errors",
			patterns: []string{"x", "a[", "b[-]", "z"},
			wantErr:  "syntax error in pattern: \"a[\"\nsyntax error in pattern: \"b[-]\"",
		},
		{
			name:     "include+exclude",
			patterns: []string{"!*x", "abc*", "def", "!*y*"},
			want:     []string{"abc123", "abc", "def"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotErrMsg := ""
			got, gotErr := filter.SelectNames(input, test.patterns)
			if gotErr != nil {
				gotErrMsg = gotErr.Error()
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("unexpected result, +got -want:\n%v", diff)
			}
			if diff := cmp.Diff(test.wantErr, gotErrMsg); diff != "" {
				t.Errorf("unexpected error, +got -want:\n%v", diff)
			}
		})
	}
}
