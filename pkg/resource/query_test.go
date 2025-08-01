package resource

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNodePathString(t *testing.T) {
	tests := map[string]struct {
		input Query
		want  string
	}{
		`nil`: {
			nil,
			"",
		},
		`empty`: {
			Query{},
			"",
		},
		`simple`: {
			Query{"a", "b", "c"},
			"a.b.c",
		},
		`escaped`: {
			Query{"a.b", "c"},
			"[a.b].c",
		},
		`filter`: {
			Query{"[a=b]", "c"},
			"[a=b].c",
		},
		`filter-with-dot`: {
			Query{"[a=b.1]", "c"},
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
		Got     Query `yaml:"got"`
		want    Query
		wantErr bool
	}{
		{
			name:  "list",
			input: `got: [ a, b, c ]`,
			want:  Query{"a", "b", "c"},
		},
		{
			name:  "text",
			input: `got: a.b.c`,
			want:  Query{"a", "b", "c"},
		},
		{
			name:  "condition",
			input: `got: a.[b=1].c`,
			want:  Query{"a", "[b=1]", "c"},
		},
		{
			name:  "glob",
			input: `got: a.*.b`,
			want:  Query{"a", "*", "b"},
		},
		{
			name:  "mixed",
			input: `got: a[b=1].*.[c=2]d`,
			want:  Query{"a[b=1]", "*", "[c=2]d"},
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
		input Query
		wantP Query
		wantC []string
		wantE bool
	}{
		{
			name:  "no-conditions",
			input: Query{"a", "b", "c"},
			wantP: Query{"a", "b", "c"},
			wantC: []string{"", "", ""},
		},
		{
			name:  "prefix-only",
			input: Query{"[a=1]b", "*", "[c=2]d"},
			wantP: Query{"b", "*", "d"},
			wantC: []string{"[a=1]", "", "[c=2]"},
		},
		{
			name:  "glob",
			input: Query{"a", "[b=1]", "c"},
			wantP: Query{"a", "*", "c"},
			wantC: []string{"", "", "[b=1]"},
		},
		{
			name:  "overflow",
			input: Query{"a", "b[c=1,d=2]"},
			wantP: Query{"a", "b", "*"},
			wantC: []string{"", "", "[c=1,d=2]"},
		},
		{
			name:  "merge",
			input: Query{"a[b=1]", "[c=2]d"},
			wantP: Query{"a", "d"},
			wantC: []string{"", "[b=1,c=2]"},
		},
		{
			name:  "error",
			input: Query{"a[b=1]", "[c=2]", "d"},
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

func TestQueriesAdd(t *testing.T) {
	tests := []struct {
		name  string
		input []Query
		want  *Queries[int]
	}{
		{
			name: "no-overlap",
			input: []Query{
				{"x", "y", "z"},
				{"a", "b", "c"},
			},
			want: &Queries[int]{
				prefix: Query{},
				queries: []*Queries[int]{
					{prefix: Query{"x", "y", "z"}, meta: 1},
					{prefix: Query{"a", "b", "c"}, meta: 2},
				},
			},
		},
		{
			name: "override",
			input: []Query{
				{"a", "b", "c"},
				{"a", "b", "c"},
			},
			want: &Queries[int]{
				prefix: Query{"a", "b", "c"},
				meta:   2,
			},
		},
		{
			name: "split-mid",
			input: []Query{
				{"a", "b", "1"},
				{"a", "b", "2"},
			},
			want: &Queries[int]{
				prefix: Query{"a", "b"},
				queries: []*Queries[int]{
					{prefix: Query{"1"}, meta: 1},
					{prefix: Query{"2"}, meta: 2},
				},
			},
		},
		{
			name: "split-end",
			input: []Query{
				{"a", "b"},
				{"a", "b", "c"},
			},
			want: &Queries[int]{
				prefix: Query{"a", "b"},
				queries: []*Queries[int]{
					{prefix: Query{}, meta: 1},
					{prefix: Query{"c"}, meta: 2},
				},
			},
		},
		{
			name: "split-end2",
			input: []Query{
				{"a", "b", "c"},
				{"a", "b"},
			},
			want: &Queries[int]{
				prefix: Query{"a", "b"},
				queries: []*Queries[int]{
					{prefix: Query{"c"}, meta: 1},
					{prefix: Query{}, meta: 2},
				},
			},
		},
		{
			name: "split-deep",
			input: []Query{
				{"a"},
				{"a", "b"},
				{"a", "c", "d"},
			},
			want: &Queries[int]{
				prefix: Query{"a"},
				queries: []*Queries[int]{
					{
						prefix: Query{},
						meta:   1,
					},
					{
						prefix: Query{"b"},
						meta:   2,
					},
					{
						prefix: Query{"c", "d"},
						meta:   3,
					},
				},
			},
		},
		{
			name: "deployment",
			input: []Query{
				{"metadata", "name"},
				{"spec", "template", "spec", "containers", "*", "name"},
				{"spec", "template", "spec", "containers", "*", "image"},
				{"spec", "template", "spec", "containers", "*", "args", "*"},
				{"spec", "template", "spec", "containers", "*", "env", "*", "name"},
				{"spec", "template", "spec", "containers", "*", "env", "*", "value"},
			},
			want: &Queries[int]{
				prefix: Query{},
				queries: []*Queries[int]{
					{
						prefix: Query{"metadata", "name"},
						meta:   1,
					},
					{
						prefix: Query{"spec", "template", "spec", "containers", "*"},
						queries: []*Queries[int]{
							{
								prefix: Query{"name"},
								meta:   2,
							},
							{
								prefix: Query{"image"},
								meta:   3,
							},
							{
								prefix: Query{"args", "*"},
								meta:   4,
							},
							{
								prefix: Query{"env", "*"},
								queries: []*Queries[int]{
									{
										prefix: Query{"name"},
										meta:   5,
									},
									{
										prefix: Query{"value"},
										meta:   6,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	allowUnexported := cmp.AllowUnexported(Queries[int]{})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			qq := &Queries[int]{}
			for i, q := range test.input {
				qq.Add(q, i+1)
			}

			if diff := cmp.Diff(test.want, qq, allowUnexported); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestQueriesScan(t *testing.T) {
	node := yaml.MustParse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-name
spec:
  template:
    spec:
      containers:
      - name: app-container
        image: app-image
        args: ["arg1","arg2"]
        env:
        - name: A
          value: 1
        - name: B
          value: 2
      - name: sidecar-container
        image: sidecar-image
        env:
        - name: C
          value: 3
`)

	queries := &Queries[string]{
		prefix: Query{},
		queries: []*Queries[string]{
			{
				prefix: Query{"metadata", "name"},
				meta:   "name",
			},
			{
				prefix: Query{"spec", "template", "spec", "containers", "*"},
				queries: []*Queries[string]{
					{
						prefix: Query{"name"},
						meta:   "container-name",
					},
					{
						prefix: Query{"image"},
						meta:   "container-image",
					},
					{
						prefix: Query{"args", "*"},
						meta:   "arg",
					},
					{
						prefix: Query{"env", "*"},
						queries: []*Queries[string]{
							{
								prefix: Query{"name"},
								meta:   "env-name",
							},
							{
								prefix: Query{"value"},
								meta:   "env-value",
							},
						},
					},
				},
			},
		},
	}
	want := []string{
		"name:app-name",
		"container-name:app-container",
		"container-image:app-image",
		`arg:"arg1"`,
		`arg:"arg2"`,
		"env-name:A",
		"env-value:1",
		"env-name:B",
		"env-value:2",
		"container-name:sidecar-container",
		"container-image:sidecar-image",
		"env-name:C",
		"env-value:3",
	}

	got := []string{}
	for k, v := range queries.Scan(node) {
		vs, _ := v.String()
		got = append(got, fmt.Sprintf("%v:%s", k, strings.TrimSpace(vs)))
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("-want +got:\n%s", diff)
	}
}
