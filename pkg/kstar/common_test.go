package kstar

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.starlark.net/starlark"
)

var cmpOpts = []cmp.Option{
	cmp.AllowUnexported(starlark.List{}),
	cmpopts.IgnoreFields(starlark.List{}, "frozen"),
}
