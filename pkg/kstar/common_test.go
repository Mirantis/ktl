package kstar

import (
	"testing"

	"github.com/go-openapi/jsonpointer"
	"github.com/go-openapi/jsonreference"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

const None = starlark.None

type StringDict = starlark.StringDict

var (
	FromStringDict = starlarkstruct.FromStringDict

	commonCmpOpts = []cmp.Option{
		cmp.AllowUnexported(matchPattern{}),
		cmp.AllowUnexported(MappingNode{}),
		cmp.AllowUnexported(ScalarNode{}),
		cmp.AllowUnexported(jsonreference.Ref{}),
		cmp.AllowUnexported(jsonpointer.Pointer{}),
		cmp.AllowUnexported(starlark.Int{}),
		cmp.AllowUnexported(starlark.List{}),
		cmpopts.IgnoreFields(starlark.List{}, "frozen"),
	}
)

type wantErr bool

func (expected wantErr) check(t *testing.T, err error) (stop bool) {
	t.Helper()

	switch {
	case bool(expected) && err == nil:
		t.Fatal("want error, got none")
	case bool(expected) && err != nil:
		t.Logf("got expected error: %v", err)
		return true
	case !bool(expected) && err != nil:
		t.Fatalf("got error: %v, want none", err)
	}

	return false
}

type wantPanic bool

func (expected wantPanic) recover(t *testing.T) {
	t.Helper()

	if !expected {
		// let it panic
		return
	}

	if err := recover(); err != nil {
		t.Logf("got expected panic: %v", err)
	}
}

func (expected wantPanic) check(t *testing.T) (stop bool) {
	if bool(expected) {
		t.Fatal("want panic, got none")
	}

	return false
}

func runStarlarkTest(t *testing.T, name, script string, input StringDict, wantPanic wantPanic, wantErr wantErr, validate func(t *testing.T, gotAll StringDict)) {
	t.Run(name, func(t *testing.T) {
		defer wantPanic.recover(t)

		opts := &syntax.FileOptions{
			TopLevelControl: true,
		}

		thread := &starlark.Thread{
			Name: name,
			Print: func(_ *starlark.Thread, msg string) {
				t.Logf("starlark output: %s", msg)
			},
		}

		result, err := starlark.ExecFileOptions(
			opts,
			thread,
			name,
			script,
			input,
		)

		if wantPanic.check(t) {
			return
		}

		if wantErr.check(t, err) {
			return
		}

		validate(t, result)
	})
}
