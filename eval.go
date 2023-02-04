package gokonfi

import (
	"fmt"
	"log"
	"strconv"

	"github.com/dnswlt/gokonfi/token"
)

type Val interface {
	fmt.Stringer
	Bool() bool // TODO: make this a regular function
	valImpl()
}

// Marker interface for "lazy" values, which are used during evaluation.
// They can be one of
// - a fully evaluated Val
// - an Expr that still needs to be evaluated
type LazyVal interface {
	lazyImpl()
}

// An unevaluated expression used as a LazyVal.
type LazyExpr struct {
	E Expr
}

// A fully evaluated Val used as a LazyVal.
type FullyEvaluated struct {
	V Val
}

func (l *LazyExpr) lazyImpl()       {}
func (l *FullyEvaluated) lazyImpl() {}

// Ctx is a chained evaluation context containing lazy evaluated values for
// all variables that are in scope.
type Ctx struct {
	env    map[string]LazyVal // let vars and fields of the current record / module.
	active map[string]bool    // To detect evaluation cycles
	parent *Ctx               // Parent context (e.g. of the parent record).
}

func NewCtx() *Ctx {
	return &Ctx{env: make(map[string]LazyVal), active: make(map[string]bool), parent: nil}
}

func ChildCtx(parent *Ctx) *Ctx {
	return &Ctx{env: make(map[string]LazyVal), active: make(map[string]bool), parent: parent}
}

func GlobalCtx() *Ctx {
	ctx := NewCtx()
	for _, builtin := range builtinFunctions {
		ctx.Store(builtin.Name, builtin)
	}
	return ctx
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
type CallableVal interface {
	Call(args []Val, ctx *Ctx) (Val, error)
}
type NativeFuncVal struct {
	F     func([]Val) (Val, error)
	Name  string
	Arity int
}
type FuncExprVal struct {
	F   *FuncExpr
	ctx *Ctx // "Closure": Context captured at function declaration
}

func (f *NativeFuncVal) Call(args []Val, ctx *Ctx) (Val, error) {
	// Negative arity means "accept any args".
	if f.Arity >= 0 && len(args) != f.Arity {
		return nil, fmt.Errorf("wrong number of arguments for %s: got %d want %d", f.Name, len(args), f.Arity)
	}
	return f.F(args)
}

func (f *FuncExprVal) Call(args []Val, ctx *Ctx) (Val, error) {
	arity := len(f.F.Params)
	if len(args) != arity {
		return nil, fmt.Errorf("wrong number of arguments for %s: got %d want %d", f.String(), len(args), arity)
	}
	fctx := ChildCtx(f.ctx)
	for i, p := range f.F.Params {
		fctx.Store(p.Name, args[i])
	}
	return Eval(f.F.Body, fctx)
}

func (v IntVal) valImpl()        {}
func (v DoubleVal) valImpl()     {}
func (v BoolVal) valImpl()       {}
func (v StringVal) valImpl()     {}
func (v NilVal) valImpl()        {}
func (v *RecVal) valImpl()       {}
func (v NativeFuncVal) valImpl() {}
func (v FuncExprVal) valImpl()   {}

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
func (r *NativeFuncVal) Bool() bool {
	return true
}
func (r *FuncExprVal) Bool() bool {
	return true
}

func (x IntVal) String() string {
	return strconv.FormatInt(int64(x), 10)
}
func (v DoubleVal) String() string {
	return strconv.FormatFloat(float64(v), 'f', -1, 64)
}
func (b BoolVal) String() string {
	return strconv.FormatBool(bool(b))
}
func (s StringVal) String() string {
	return string(s)
}
func (s NilVal) String() string {
	return "nil"
}
func (r *RecVal) String() string {
	return "<rec>"
}
func (f *NativeFuncVal) String() string {
	return fmt.Sprintf("<builtin %s>", f.Name)
}

func (f *FuncExprVal) String() string {
	return fmt.Sprintf("<func @%d:%d>", f.F.Pos(), f.F.End())
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
func LessThan(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return BoolVal(u < v), nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return BoolVal(u < v), nil
		}
	}
	return nil, fmt.Errorf("incompatible types for <: %T and %T", x, y)
}
func LessEq(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return BoolVal(u <= v), nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return BoolVal(u <= v), nil
		}
	}
	return nil, fmt.Errorf("incompatible types for <: %T and %T", x, y)
}
func GreaterThan(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return BoolVal(u > v), nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return BoolVal(u > v), nil
		}
	}
	return nil, fmt.Errorf("incompatible types for <: %T and %T", x, y)
}
func GreaterEq(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return BoolVal(u >= v), nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return BoolVal(u >= v), nil
		}
	}
	return nil, fmt.Errorf("incompatible types for <: %T and %T", x, y)
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

func UnaryNot(x Val) (Val, error) {
	return BoolVal(!x.Bool()), nil
}

func UnaryOp(x Val, op token.TokenType) (Val, error) {
	switch op {
	case token.Minus:
		return UnaryMinus(x)
	case token.Not:
		return UnaryNot(x)
	}
	return nil, fmt.Errorf("invalid unary operator '%v'", op)
}

func BinaryOp(x, y Val, op token.TokenType) (Val, error) {
	switch op {
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
	case token.LessThan:
		return LessThan(x, y)
	case token.LessEq:
		return LessEq(x, y)
	case token.GreaterThan:
		return GreaterThan(x, y)
	case token.GreaterEq:
		return GreaterEq(x, y)
	case token.Merge:
		return MergeRecords(x, y)
	}
	return nil, fmt.Errorf("invalid binary operator '%v'", op)
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
		r, err := UnaryOp(x, e.Op)
		if err != nil {
			return nil, &EvalError{pos: e.Pos(), msg: err.Error()}
		}
		return r, nil
	case *BinaryExpr:
		x, err := Eval(e.X, ctx)
		if err != nil {
			return nil, err
		}
		y, err := Eval(e.Y, ctx)
		if err != nil {
			return nil, err
		}
		r, err := BinaryOp(x, y, e.Op)
		if err != nil {
			return nil, &EvalError{pos: e.Pos(), msg: err.Error()}
		}
		return r, nil
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
	case *CallExpr:
		fe, err := Eval(e.Func, ctx)
		if err != nil {
			return nil, err
		}
		f, ok := fe.(CallableVal)
		if !ok {
			return nil, &EvalError{pos: e.Func.Pos(), msg: fmt.Sprintf("Type %T is not callable", fe)}
		}
		args := make([]Val, len(e.Args))
		for i, arg := range e.Args {
			val, err := Eval(arg, ctx)
			if err != nil {
				return nil, err
			}
			args[i] = val
		}
		res, err := f.Call(args, ctx)
		if err == nil {
			return res, nil
		}
		// Propagate EvalErrors, wrap all others to retain the source location.
		if _, ok := err.(*EvalError); ok {
			return nil, err
		}
		return nil, &EvalError{pos: e.Func.Pos(), msg: err.Error()}
	case *FuncExpr:
		return &FuncExprVal{F: e, ctx: ctx}, nil
	case *ConditionalExpr:
		cond, err := Eval(e.Cond, ctx)
		if err != nil {
			return nil, err
		}
		// Only evaluate one of the two branches.
		if cond.Bool() {
			return Eval(e.X, ctx)
		}
		return Eval(e.Y, ctx)
	}
	return nil, &EvalError{pos: expr.Pos(), msg: "not implemented"}
}

func MergeRecords(x, y Val) (Val, error) {
	u, ok := x.(*RecVal)
	if !ok {
		return nil, fmt.Errorf("cannot merge lhs of type %T", x)
	}
	v, ok := y.(*RecVal)
	if !ok {
		return nil, fmt.Errorf("cannot merge rhs of type %T", y)
	}
	r := NewRec()
	if err := MergeRecVal(u, v, r); err != nil {
		return nil, err
	}
	return r, nil
}

func MergeRecVal(x, y, r *RecVal) error {
	// Copy fields only in x.
	for f, v := range x.Fields {
		if _, ok := y.Fields[f]; !ok {
			r.SetField(f, v)
		}
	}
	// Copy fields only in y and merge common fields.
	for f, v := range y.Fields {
		if _, ok := x.Fields[f]; !ok {
			// Unique field of y.
			r.SetField(f, v)
		} else {
			// Common field.
			if cy, ok := v.(*RecVal); !ok {
				// y field is not a record, just take the value from y.
				r.SetField(f, v)
			} else if cx, ok := x.Fields[f].(*RecVal); ok {
				// x's field is a record, too: recurse.
				cr := NewRec()
				r.SetField(f, cr)
				MergeRecVal(cx, cy, cr)
			} else {
				// x field is not a record, again just take the value from y.
				r.SetField(f, v)
			}
		}
	}
	return nil
}
