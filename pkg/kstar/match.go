package kstar

import (
	"fmt"
	"path"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type matchPattern string

func (match matchPattern) String() string {
	return fmt.Sprintf("match(%s)", syntax.Quote(string(match), false))
}

var matchPatternType = starlark.String("").Type()

func (match matchPattern) Type() string {
	return matchPatternType
}

func (match matchPattern) Freeze() {
}

func (match matchPattern) Truth() starlark.Bool {
	return starlark.String(match).Truth()
}

func (match matchPattern) Hash() (uint32, error) {
	return starlark.String(match).Hash()
}

func newMatchPattern(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pattern string
	starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &pattern)

	if _, err := path.Match(pattern, ""); err != nil {
		return nil, fmt.Errorf("invalid match pattern: %w", err)
	}

	return matchPattern(pattern), nil
}
