package kstar

import (
	"fmt"
	"path"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

const fnMatchPattern = "match"

type matchPattern string

var (
	_ starlark.Value = new(matchPattern)
)

func (match matchPattern) String() string {
	return fmt.Sprintf("%s(%s)", fnMatchPattern, syntax.Quote(string(match), false))
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

func (match matchPattern) Name() string {
	return fnMatchPattern
}

func (match matchPattern) apply(value starlark.Value) (starlark.Value, error) {
	switch v := value.(type) {
	case starlark.Iterable:
		return match.applyIterable(v)
	default:
		return match.applySingle(value)
	}
}

func (match matchPattern) applyIterable(value starlark.Iterable) (starlark.Value, error) {
	results := starlark.NewList(nil)

	iter := value.Iterate()
	defer iter.Done()

	var item starlark.Value
	for iter.Next(&item) {
		matched, err := match.applySingle(item)
		if err != nil {
			return nil, err
		}

		if matched == starlark.None {
			continue
		}

		err = results.Append(matched)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", match.String(), err)
		}
	}

	return results, nil
}

func (match matchPattern) applySingle(value starlark.Value) (starlark.Value, error) {
	switch v := value.(type) {
	case starlark.String:
		ok, _ := path.Match(string(match), v.GoString())
		if ok {
			return value, nil
		}
		return starlark.None, nil
	default:
		return nil, fmt.Errorf("%s: type %s not supported", fnMatchPattern, value.Type())
	}
}

func newMatchPattern(_ *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var pattern string
	var value starlark.Value

	err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &pattern, &value)
	if err != nil {
		return nil, err
	}

	_, err = path.Match(pattern, "")
	if err != nil {
		return nil, fmt.Errorf("%s: %w (%q)", fn.Name(), err, pattern)
	}

	match := matchPattern(pattern)

	if len(args) == 1 {
		return match, nil
	}

	return match.apply(value)
}
