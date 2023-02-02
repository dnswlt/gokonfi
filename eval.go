package gokonfi

import (
	"fmt"

	"github.com/dnswlt/gokonfi/token"
)

type Val interface {
	Bool() bool
	valImpl()
}

type Ctx struct {
	letVars map[string]Val
	recExpr *RecExpr
	rec     *RecVal
	parent  *Ctx
}

func NewCtx() *Ctx {
	return &Ctx{letVars: make(map[string]Val), recExpr: nil, rec: nil, parent: nil}
}

type EvalError struct {
	pos token.Pos
	msg string
}

func (e *EvalError) Error() string {
	return fmt.Sprintf("EvalError: %s at position %d", e.msg, e.pos)
}

type RecVal struct {
	Fields map[string]Val
}

type IntVal int64
type DoubleVal float64
type BoolVal bool
type StringVal string
type NilVal struct{}

func (v IntVal) valImpl()    {}
func (v DoubleVal) valImpl() {}
func (v BoolVal) valImpl()   {}
func (v StringVal) valImpl() {}
func (v NilVal) valImpl()    {}
func (v *RecVal) valImpl()   {}

func (x IntVal) Bool() bool {
	return x != 0
}

func (v DoubleVal) Bool() bool {
	return v != 0
}
func (b BoolVal) Bool() bool {
	return bool(b)
}
func (s StringVal) Bool() bool {
	return len(s) > 0
}
func (s NilVal) Bool() bool {
	return false
}
func (r *RecVal) Bool() bool {
	return len(r.Fields) > 0
}

// Binary operations on Val.

func Plus(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return u + v, nil
		}
	} else if u, ok := x.(StringVal); ok {
		if v, ok := y.(StringVal); ok {
			return u + v, nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return u + v, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for +: %T and %T", x, y)
}
func Minus(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return u - v, nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return u - v, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for -: %T and %T", x, y)
}
func Times(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return u * v, nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return u * v, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for *: %T and %T", x, y)
}
func Div(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return u / v, nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return u / v, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for /: %T and %T", x, y)
}
func Modulo(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return u % v, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for /: %T and %T", x, y)
}
func LogicalAnd(x, y Val) (Val, error) {
	return BoolVal(x.Bool() && y.Bool()), nil
}
func LogicalOr(x, y Val) (Val, error) {
	return BoolVal(x.Bool() || y.Bool()), nil
}

// Val equality is delegated to Go equality.
// This works as expected for scalar types.
// Records never compare equal.
func Equal(x, y Val) (Val, error) {
	return BoolVal(x == y), nil
}
func NotEqual(x, y Val) (Val, error) {
	return BoolVal(x != y), nil
}

// Unary operations on Val.
func UnaryMinus(x Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		return -u, nil

	} else if u, ok := x.(DoubleVal); ok {
		return -u, nil
	}
	return nil, fmt.Errorf("incompatible type for unary -: %T", x)
}

func Eval(expr Expr, ctx *Ctx) (Val, error) {
	switch e := expr.(type) {
	case *IntLiteral:
		return IntVal(e.Val), nil
	case *BoolLiteral:
		return BoolVal(e.Val), nil
	case *DoubleLiteral:
		return DoubleVal(e.Val), nil
	case *StrLiteral:
		return StringVal(e.Val), nil
	case *NilLiteral:
		return NilVal{}, nil
	case *UnaryExpr:
		x, err := Eval(e.X, ctx)
		if err != nil {
			return nil, err
		}
		switch e.Op {
		case token.Minus:
			return UnaryMinus(x)
		}
	case *BinaryExpr:
		x, err := Eval(e.X, ctx)
		if err != nil {
			return nil, err
		}
		y, err := Eval(e.Y, ctx)
		if err != nil {
			return nil, err
		}
		switch e.Op {
		case token.Plus:
			return Plus(x, y)
		case token.Minus:
			return Minus(x, y)
		case token.Times:
			return Times(x, y)
		case token.Div:
			return Div(x, y)
		case token.Modulo:
			return Modulo(x, y)
		case token.LogicalAnd:
			return LogicalAnd(x, y)
		case token.LogicalOr:
			return LogicalOr(x, y)
		case token.Equal:
			return Equal(x, y)
		case token.NotEqual:
			return NotEqual(x, y)
			/*
				LessThan
				LessEq
				GreaterThan
				GreaterEq
			*/
		}
		return nil, &EvalError{pos: e.OpPos, msg: fmt.Sprintf("invalid binary operator: %s", e.Op)}
	case *VarExpr:
		break
	}
	return nil, &EvalError{pos: expr.Pos(), msg: "not implemented"}
}
