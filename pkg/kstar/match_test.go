package kstar

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		name   string
		expr string
		want   starlark.Value
		wantErr bool
	}{
		{
			name: "ctor",
			expr: `match("my*")`,
			want: matchPattern("my*"),
		},
		{
			name: "ctor-err",
			expr: `match("my[")`,
			wantErr: true,
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
				fmt.Sprintf("%s = %s", resultVar, test.expr),
				starlark.StringDict{
					"match": starlark.NewBuiltin("match", newMatchPattern),
				},
			)

			got := gotAll[resultVar]

			if err != nil && !test.wantErr {
				t.Fatal(err)
			}

			if err == nil && test.wantErr {
				t.Fatal("want error, got none")
			}

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}
