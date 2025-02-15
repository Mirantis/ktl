package filters_test

import (
	"log/slog"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/filters"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestClearAll(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	tests := []struct {
		name    string
		path    types.NodePath
		input   string
		want    string
		wantErr bool
	}{
		{
			name: "no-conditions",
			path: types.NodePath{"a", "b"},
			input: `a:
  b: 1
  c: 2
`,
			want: `a:
  c: 2
`,
		},
		{
			name: "tail-condition",
			path: types.NodePath{"a", "b[=1]"},
			input: `a:
  b: 1
  c: 2
`,
			want: `a:
  c: 2
`,
		},
		{
			name: "tail-condition-no-match",
			path: types.NodePath{"a", "b[=0]"},
			input: `a:
  b: 1
  c: 2
`,
			want: `a:
  b: 1
  c: 2
`,
		},
		{
			name: "tail-condition-no-match",
			path: types.NodePath{"a", "b[=0]"},
			input: `a:
  b: 1
  c: 2
`,
			want: `a:
  b: 1
  c: 2
`,
		},
		{
			name: "list",
			path: types.NodePath{"a", "b[=[\"ABcD\"]]"},
			input: `a:
  b: [ "ABcD" ]
  c: 2
`,
			want: `a:
  c: 2
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := yaml.MustParse(test.input)
			f, err := filters.ClearAll(test.path)
			if err != nil {
				t.Fatalf("invalid filter: %v", err)
			}
			got, err := f.Filter(input)
			if err != nil && test.wantErr {
				return
			}
			if err != nil {
				t.Fatalf("want no error, got: %v", err)
			}
			if test.wantErr {
				t.Fatalf("want error, got none")
			}
			gotText, err := got.String()
			if err != nil {
				t.Fatalf("invalid result yaml: %v", err)
			}
			if diff := cmp.Diff(test.want, gotText); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}
