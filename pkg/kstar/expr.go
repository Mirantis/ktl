package kstar

import (
	"fmt"
	"slices"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

const nodeExprType = "NodeExpression"

type nodeExprTarget interface {
	starlark.Value
	starlark.HasSetField
	starlark.HasSetKey
	starlark.HasBinary

	clone() nodeExprTarget
	exprOp(op syntax.Token, value starlark.Value, side starlark.Side) nodeExprOp
}

type nodeExprOp func(target nodeExprTarget) (nodeExprTarget, error)

type nodeExpr struct {
	target nodeExprTarget
	ops    []nodeExprOp
}

func (expr *nodeExpr) materialize() (nodeExprTarget, error) {
	if len(expr.ops) == 0 {
		return expr.target, nil
	}

	node := expr.target.clone()

	for _, op := range expr.ops {
		var err error
		node, err = op(node)

		if err != nil {
			return nil, fmt.Errorf("unable to materialize node expression: %w", err)
		}
	}

	expr.target = node
	expr.ops = nil

	return expr.target, nil
}

func (expr *nodeExpr) String() string {
	if len(expr.ops) == 0 {
		return expr.target.String()
	}

	return nodeExprType + "()"
}

func (expr *nodeExpr) Type() string {
	if len(expr.ops) == 0 {
		return expr.target.Type()
	}

	return nodeExprType
}

func (expr *nodeExpr) Freeze() {
	expr.target.Freeze()
}

func (expr *nodeExpr) Truth() starlark.Bool {
	node, err := expr.materialize()
	if err != nil {
		return false
	}

	return node.Truth()
}

func (expr *nodeExpr) Hash() (uint32, error) {
	node, err := expr.materialize()
	if err != nil {
		return 0, err
	}

	return node.Hash()
}

func (expr *nodeExpr) Attr(name string) (starlark.Value, error) {
	node, err := expr.materialize()
	if err != nil {
		return nil, err
	}

	return node.Attr(name)
}

func (expr *nodeExpr) AttrNames() []string {
	node, err := expr.materialize()
	if err != nil {
		return nil
	}

	return node.AttrNames()
}

func (expr *nodeExpr) SetField(name string, value starlark.Value) error {
	node, err := expr.materialize()
	if err != nil {
		return err
	}

	return node.SetField(name, value)
}

func (expr *nodeExpr) Get(key starlark.Value) (_ starlark.Value, found bool, _ error) {
	node, err := expr.materialize()
	if err != nil {
		return nil, false, err
	}

	return node.Get(key)
}

func (expr *nodeExpr) SetKey(key, value starlark.Value) error {
	node, err := expr.materialize()
	if err != nil {
		return err
	}

	return node.SetKey(key, value)
}

func (expr *nodeExpr) Binary(op syntax.Token, value starlark.Value, side starlark.Side) (starlark.Value, error) {
	exprOp := expr.target.exprOp(op, value, side)
	if exprOp == nil {
		return nil, nil
	}

	return &nodeExpr{
		target: expr.target,
		ops:    slices.Concat(expr.ops, []nodeExprOp{exprOp}),
	}, nil
}
