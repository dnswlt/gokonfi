package gokonfi

import (
	"fmt"
	"log"

	"github.com/dnswlt/gokonfi/token"
)

type Val interface {
	Bool() bool // TODO: make this a regular function
	valImpl()
}

// Marker interface for "lazy" values. Those can be one of
// - a fully evaluated Val
// - an Expr that still needs to be evaluated
type LazyVal interface {
	lazyImpl()
}

type LazyExpr struct {
	E Expr
}
type FullyEvaluated struct {
	V Val
}

func (l *LazyExpr) lazyImpl()       {}
func (l *FullyEvaluated) lazyImpl() {}

type Ctx struct {
	env    map[string]LazyVal // let vars and fields of the current record / module.
	active map[string]bool    // To detect evaluation cycles
	parent *Ctx
}

func NewCtx() *Ctx {
	return &Ctx{env: make(map[string]LazyVal), active: make(map[string]bool), parent: nil}
}

func ChildCtx(parent *Ctx) *Ctx {
	return &Ctx{env: make(map[string]LazyVal), active: make(map[string]bool), parent: parent}
}

func (ctx *Ctx) Lookup(v string) (LazyVal, *Ctx) {
	c := ctx
	for c != nil {
		if val, ok := c.env[v]; ok {
			return val, c
		}
		c = c.parent
	}
	return nil, nil // Not found
}

func (ctx *Ctx) IsActive(v string) bool {
	return ctx.active[v]
}

func (ctx *Ctx) SetActive(v string) {
	ctx.active[v] = true
}

func (ctx *Ctx) Store(v string, val Val) {
	ctx.env[v] = &FullyEvaluated{val}
	// Once a value was stored, it's no longer actively being computed.
	delete(ctx.active, v)
}

func (ctx *Ctx) StoreExpr(v string, expr Expr) {
	ctx.env[v] = &LazyExpr{E: expr}
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

func NewRec() *RecVal {
	return &RecVal{Fields: make(map[string]Val)}
}

func (r *RecVal) SetField(field string, val Val) {
	r.Fields[field] = val
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
		lval, vctx := ctx.Lookup(e.Name)
		if lval == nil {
			return nil, &EvalError{pos: e.Pos(), msg: fmt.Sprintf("Unbound variable %s", e.Name)}
		}
		switch lv := lval.(type) {
		case *FullyEvaluated:
			return lv.V, nil
		case *LazyExpr:
			if vctx.IsActive(e.Name) {
				return nil, &EvalError{pos: e.Pos(), msg: "Cyclic variable dependencies detected"}
			}
			vctx.SetActive(e.Name)
			v, err := Eval(lv.E, vctx)
			if err != nil {
				return nil, err
			}
			vctx.Store(e.Name, v)
			return v, nil
		default:
			log.Fatalf("Unhandled type for LazyVal: %T", lval)
		}
	case *RecExpr:
		rctx := ChildCtx(ctx)
		// Prepare context by storing lazy expressions of all fields.
		for _, lv := range e.LetVars {
			rctx.StoreExpr(lv.Name, lv.Val)
		}
		for _, f := range e.Fields {
			rctx.StoreExpr(f.Name, f.Val)
		}
		// Evaluate all fields.
		for _, lv := range e.LetVars {
			rctx.SetActive(lv.Name)
			v, err := Eval(lv.Val, rctx)
			if err != nil {
				return nil, err
			}
			rctx.Store(lv.Name, v)
		}
		rec := NewRec()
		for _, f := range e.Fields {
			rctx.SetActive(f.Name)
			v, err := Eval(f.Val, rctx)
			if err != nil {
				return nil, err
			}
			rctx.Store(f.Name, v)
			rec.SetField(f.Name, v)
		}
		return rec, nil
	case *FieldAcc:
		v, err := Eval(e.X, ctx)
		if err != nil {
			return nil, err
		}
		switch r := v.(type) {
		case *RecVal:
			if v, ok := r.Fields[e.Name]; ok {
				return v, nil
			}
			// TODO: Add DotPos to FieldAcc.
			return nil, &EvalError{pos: e.End(), msg: fmt.Sprintf("Record has no field '%s'", e.Name)}
		default:
			return nil, &EvalError{pos: e.End(), msg: fmt.Sprintf("Cannot access .%s on type %T", e.Name, e)}
		}
	}
	return nil, &EvalError{pos: expr.Pos(), msg: "not implemented"}
}
