package kstar

import (
	"fmt"
	"path"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

const fnMatchPattern = "match"

type matchPattern struct {
	pattern string
	inverse bool
}

var (
	_ starlark.Value     = new(matchPattern)
	_ starlark.HasBinary = new(matchPattern)
	_ starlark.HasUnary  = new(matchPattern)
)

func (match *matchPattern) String() string {
	inverse := ""
	if match.inverse {
		inverse = "~"
	}
	return fmt.Sprintf("%s%s(%s)", inverse, fnMatchPattern, syntax.Quote(match.pattern, false))
}

func (match *matchPattern) Type() string {
	return fnMatchPattern
}

func (match *matchPattern) Freeze() {
}

func (match *matchPattern) Truth() starlark.Bool {
	return starlark.String(match.pattern).Truth()
}

func (match *matchPattern) Hash() (uint32, error) {
	inverse := ""
	if match.inverse {
		inverse = "~"
	}
	return starlark.String(inverse + match.pattern).Hash()
}

func (match *matchPattern) Name() string {
	return fnMatchPattern
}

func (match *matchPattern) Binary(op syntax.Token, value starlark.Value, _ starlark.Side) (starlark.Value, error) {
	if op != syntax.IN {
		return nil, nil
	}

	return match.apply(value)
}

func (match *matchPattern) Unary(op syntax.Token) (starlark.Value, error) {
	if op != syntax.TILDE {
		return nil, nil
	}

	return &matchPattern{match.pattern, !match.inverse}, nil
}

func (match *matchPattern) apply(value starlark.Value) (starlark.Value, error) {
	switch v := value.(type) {
	case starlark.Iterable:
		return match.applyIterable(v)
	default:
		return match.applySingle(value)
	}
}

func (match *matchPattern) applyIterable(value starlark.Iterable) (starlark.Value, error) {
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

func (match *matchPattern) applySingle(value starlark.Value) (starlark.Value, error) {
	switch v := value.(type) {
	case starlark.String:
		ok, _ := path.Match(match.pattern, v.GoString())
		if ok != match.inverse {
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

	match := &matchPattern{pattern, false}

	if len(args) == 1 {
		return match, nil
	}

	return match.apply(value)
}
