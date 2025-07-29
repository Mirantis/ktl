package kstar

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.starlark.net/starlark"
)

var cmpOpts = []cmp.Option{
	cmp.AllowUnexported(matchPattern{}),
	cmp.AllowUnexported(starlark.Int{}),
	cmp.AllowUnexported(starlark.List{}),
	cmpopts.IgnoreFields(starlark.List{}, "frozen"),
}

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
