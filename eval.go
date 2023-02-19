package gokonfi

import (
	"fmt"
	"log"
	"path"
	"strconv"

	"github.com/dnswlt/gokonfi/token"
)

type Val interface {
	fmt.Stringer
	Bool() bool
	Typ() *Typ
	valImpl()
}

// Marker interface for "lazy" values, which are used during evaluation.
// They can be one of
// - a fully evaluated Val
// - an Expr that still needs to be evaluated
type lazyVal interface {
	lazyImpl()
}

// An unevaluated expression used as a LazyVal.
type lazyExpr struct {
	expr Expr
}

// A fully evaluated Val used as a LazyVal.
type fullyEvaluated struct {
	val Val
}

func (l *lazyExpr) lazyImpl()       {}
func (l *fullyEvaluated) lazyImpl() {}

type Ctx struct {
	vars   *varCtx
	global *globalCtx
}

// LocalCtx is a chained evaluation context containing lazy evaluated values for
// all variables that are in scope.
type varCtx struct {
	env    map[string]lazyVal // let vars and fields of the current record / module.
	active map[string]bool    // Variables currently under evaluation (to detect evaluation cycles).
	parent *varCtx            // Parent context (e.g. of the parent record).
}

type globalCtx struct {
	fileset   *token.FileSet           // The set of files loaded thus far or currently being loaded
	types     map[string]*Typ          // Known types
	modules   map[string]*loadedModule // Already loaded modules, keyed by File.Name().
	filestack []string                 // Stack of current working directories.
}

type loadedModule struct {
	file *token.File
	body Val
}

func (m *loadedModule) Body() Val {
	return m.body
}

func EmptyCtx() *Ctx {
	return &Ctx{
		vars: &varCtx{
			env:    make(map[string]lazyVal),
			active: make(map[string]bool),
			parent: nil,
		},
		global: &globalCtx{
			fileset: token.NewFileSet(),
			types:   make(map[string]*Typ),
			modules: make(map[string]*loadedModule),
		},
	}
}

func ChildCtx(parent *Ctx) *Ctx {
	return &Ctx{
		vars: &varCtx{
			env:    make(map[string]lazyVal),
			active: make(map[string]bool),
			parent: parent.vars},
		global: parent.global,
	}
}

func GlobalCtx() *Ctx {
	ctx := EmptyCtx()
	for _, builtin := range builtinFunctions {
		ctx.store(builtin.Name, builtin)
	}
	for _, typ := range builtinTypes {
		if typ.IsUnit() {
			for _, unit := range typ.Units {
				ctx.defineUnit(unit.Name, typ)
			}
		} else {
			ctx.defineType(typ.Id, typ)
		}
	}
	return ctx
}

func (ctx *Ctx) dropLocals() *Ctx {
	// The last varCtx in the chain contains the global variables.
	l := ctx.vars
	for l.parent != nil {
		l = l.parent
	}
	return &Ctx{global: ctx.global, vars: l}
}

func (ctx *Ctx) Lookup(v string) (lazyVal, *Ctx) {
	c := ctx.vars
	for c != nil {
		if val, ok := c.env[v]; ok {
			return val, &Ctx{c, ctx.global}
		}
		c = c.parent
	}
	return nil, nil // Not found
}

func (ctx *Ctx) LookupType(typeId string) *Typ {
	if typ, ok := ctx.global.types[typeId]; ok {
		return typ
	}
	return nil // Not found
}

func (ctx *Ctx) LookupModule(name string) *loadedModule {
	if mod, ok := ctx.global.modules[name]; ok {
		return mod
	}
	return nil // Not found
}

func (ctx *Ctx) isActive(v string) bool {
	return ctx.vars.active[v]
}

func (ctx *Ctx) setActive(v string) {
	ctx.vars.active[v] = true
}

// Checks whether this particular context (ignoring its parents) has
// fully evaluated variable x. If it does, returns it, else nil.
func (ctx *Ctx) fullyEvaluated(v string) (val *fullyEvaluated, found bool) {
	if x, ok := ctx.vars.env[v]; ok {
		if val, ok := x.(*fullyEvaluated); ok {
			return val, true
		}
	}
	return nil, false
}

func (ctx *Ctx) store(v string, val Val) {
	ctx.vars.env[v] = &fullyEvaluated{val}
	// Once a value was stored, it's no longer actively being computed.
	delete(ctx.vars.active, v)
}

func (ctx *Ctx) storeExpr(v string, expr Expr) {
	ctx.vars.env[v] = &lazyExpr{expr: expr}
}

func (ctx *Ctx) storeModule(m *loadedModule) {
	ctx.global.modules[m.file.Name()] = m
}

func (ctx *Ctx) defineType(name string, typ *Typ) {
	ctx.global.types[name] = typ
}

func (ctx *Ctx) defineUnit(name string, typ *Typ) {
	ctx.global.types[name] = typ
}

func (ctx *Ctx) addFile(name string, size int) *token.File {
	return ctx.global.fileset.AddFile(name, size)
}

func (ctx *Ctx) isActiveFile(name string) bool {
	for _, f := range ctx.global.filestack {
		if f == name {
			return true
		}
	}
	return false
}

func (ctx *Ctx) pushFile(filename string) {
	ctx.global.filestack = append(ctx.global.filestack, filename)
}

func (ctx *Ctx) popFile() {
	if len(ctx.global.filestack) == 0 {
		return
	}
	ctx.global.filestack = ctx.global.filestack[:len(ctx.global.filestack)-1]
}

func (ctx *Ctx) cwd() string {
	if len(ctx.global.filestack) == 0 {
		return "."
	}
	return path.Dir(ctx.global.filestack[len(ctx.global.filestack)-1])
}

func (ctx *Ctx) fileset() *token.FileSet {
	return ctx.global.fileset
}

type EvalError struct {
	pos   token.Pos
	msg   string
	cause error
}

func (e *EvalError) Error() string {
	return fmt.Sprintf("EvalError: %s at position %d", e.msg, e.pos)
}

func (e *EvalError) Pos() token.Pos {
	return e.pos
}

func (e *EvalError) Unwrap() error {
	return e.cause
}

type RecVal struct {
	Fields     map[string]Val
	FieldTypes map[string]*Typ // Optional type annotations per field.
}

func NewRec() *RecVal {
	return &RecVal{Fields: make(map[string]Val), FieldTypes: make(map[string]*Typ)}
}

func (r *RecVal) setField(field string, val Val, typ *Typ) {
	r.Fields[field] = val
	if typ != nil {
		r.FieldTypes[field] = typ
	}
}

type ListVal struct {
	Elements []Val
}

type IntVal int64
type DoubleVal float64

// A UnitVal is used to deal with "typed numbers", numbers that have
// units such as distance in meters, number of bytes, or time duration.
// UnitVal should be used as a value type, and passed along as such.
type UnitVal struct {
	V float64 // Value, in the given unit multiple. (E.g., V == 2. and F == 1e3 represents 2e3.)
	F float64 // Multiple or submultiple of the unit.
	T *Typ    // The unit's type.
}

func (v *UnitVal) TypeId() string {
	return v.T.Id
}

type BoolVal bool
type StringVal string
type NilVal struct{}
type CallableVal interface {
	Call(args []Val, ctx *Ctx) (Val, error)
}
type NativeFuncVal struct {
	F     func([]Val, *Ctx) (Val, error)
	Name  string
	Arity int
}
type FuncExprVal struct {
	F   *FuncExpr
	ctx *Ctx // "Closure": Context captured at function declaration
}

type TypedVal struct {
	V Val
	T *Typ
}

func (v *TypedVal) TypeId() string {
	return v.T.Id
}

func (f *NativeFuncVal) Call(args []Val, ctx *Ctx) (Val, error) {
	// Negative arity means "accept any args".
	if f.Arity >= 0 && len(args) != f.Arity {
		return nil, fmt.Errorf("wrong number of arguments for %s: got %d want %d", f.Name, len(args), f.Arity)
	}
	return f.F(args, ctx)
}

func (f *FuncExprVal) Call(args []Val, _ *Ctx) (Val, error) {
	arity := len(f.F.Params)
	if len(args) != arity {
		return nil, fmt.Errorf("wrong number of arguments for %s: got %d want %d", f.String(), len(args), arity)
	}
	fctx := ChildCtx(f.ctx)
	for i, p := range f.F.Params {
		fctx.store(p.Name, args[i])
	}
	return Eval(f.F.Body, fctx)
}

func (v IntVal) valImpl()         {}
func (v DoubleVal) valImpl()      {}
func (v UnitVal) valImpl()        {}
func (v BoolVal) valImpl()        {}
func (v StringVal) valImpl()      {}
func (v NilVal) valImpl()         {}
func (v *RecVal) valImpl()        {}
func (v *ListVal) valImpl()       {}
func (v *NativeFuncVal) valImpl() {}
func (v *FuncExprVal) valImpl()   {}
func (v *TypedVal) valImpl()      {}

func (x IntVal) Bool() bool {
	return x != 0
}
func (v DoubleVal) Bool() bool {
	return v != 0
}
func (v UnitVal) Bool() bool {
	return v.V != 0
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
func (r *ListVal) Bool() bool {
	return len(r.Elements) > 0
}
func (r *NativeFuncVal) Bool() bool {
	return true
}
func (r *FuncExprVal) Bool() bool {
	return true
}
func (v TypedVal) Bool() bool {
	// User defined types for now cannot overwrite the truthiness of values.
	return v.V.Bool()
}

func (x IntVal) String() string {
	return strconv.FormatInt(int64(x), 10)
}
func (v DoubleVal) String() string {
	return strconv.FormatFloat(float64(v), 'f', -1, 64)
}
func (v UnitVal) String() string {
	if n, ok := v.T.UnitName(v.F); ok {
		f := strconv.FormatFloat(v.V, 'f', -1, 64)
		return f + "::" + n
	}
	// A UnitVal with an unknown unit is an interpreter bug.
	log.Fatalf("UnitVal %s with invalid factor %f", v.TypeId(), v.F)
	return ""
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
func (r *ListVal) String() string {
	return "<list>"
}
func (f *NativeFuncVal) String() string {
	return fmt.Sprintf("<builtin %s>", f.Name)
}
func (f *FuncExprVal) String() string {
	return fmt.Sprintf("<func @%d:%d>", f.F.Pos(), f.F.End())
}
func (v TypedVal) String() string {
	// For now, user defined types cannot override the String() method.
	return fmt.Sprintf("%s(%s)", v.T.Id, v.V.String())
}

func (x IntVal) Typ() *Typ {
	return builtinTypeInt
}
func (v DoubleVal) Typ() *Typ {
	return builtinTypeDouble
}
func (v UnitVal) Typ() *Typ {
	return v.T
}
func (b BoolVal) Typ() *Typ {
	return builtinTypeBool
}
func (s StringVal) Typ() *Typ {
	return builtinTypeString
}
func (s NilVal) Typ() *Typ {
	return builtinTypeNil
}
func (r *RecVal) Typ() *Typ {
	return builtinTypeRec
}
func (r *ListVal) Typ() *Typ {
	return builtinTypeList
}
func (r *NativeFuncVal) Typ() *Typ {
	return builtinTypeNativeFunc
}
func (r *FuncExprVal) Typ() *Typ {
	return builtinTypeFuncExpr
}
func (v TypedVal) Typ() *Typ {
	// User defined types for now cannot overwrite the truthiness of values.
	return v.T
}

// Binary operations on Val.

func plus(x, y Val) (Val, error) {
	switch u := x.(type) {
	case IntVal:
		if v, ok := y.(IntVal); ok {
			return u + v, nil
		}
	case StringVal:
		if v, ok := y.(StringVal); ok {
			return u + v, nil
		}
	case DoubleVal:
		if v, ok := y.(DoubleVal); ok {
			return u + v, nil
		}
	case UnitVal:
		if v, ok := y.(UnitVal); ok {
			if u.T != v.T {
				return nil, fmt.Errorf("incompatible unit types for +: %s and %s", u.TypeId(), v.TypeId())
			}
			if u.F == v.F {
				return UnitVal{V: u.V + v.V, F: u.F, T: u.T}, nil
			} else if u.F < v.F {
				// 1 mm(1e-3) + 1 cm(1e-2) ==> 11 mm(1e-3)
				return UnitVal{V: u.V + v.V*(v.F/u.F), F: u.F, T: u.T}, nil
			}
			// u.F > v.F
			return UnitVal{V: u.V*(u.F/v.F) + v.V, F: v.F, T: v.T}, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for +: %T and %T", x, y)
}
func minus(x, y Val) (Val, error) {
	switch u := x.(type) {
	case IntVal:
		if v, ok := y.(IntVal); ok {
			return u - v, nil
		}
	case DoubleVal:
		if v, ok := y.(DoubleVal); ok {
			return u - v, nil
		}
	case UnitVal:
		if v, ok := y.(UnitVal); ok {
			if u.T != v.T {
				return nil, fmt.Errorf("incompatible unit types for -: %s and %s", u.TypeId(), v.TypeId())
			}
			if u.F == v.F {
				return UnitVal{V: u.V - v.V, F: u.F, T: u.T}, nil
			} else if u.F < v.F {
				return UnitVal{V: u.V - v.V*(v.F/u.F), F: u.F, T: u.T}, nil
			}
			// u.F > v.F
			return UnitVal{V: u.V*(u.F/v.F) - v.V, F: v.F, T: v.T}, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for -: %T and %T", x, y)
}
func times(x, y Val) (Val, error) {
	switch u := x.(type) {
	case IntVal:
		if v, ok := y.(IntVal); ok {
			return u * v, nil
		}
		if v, ok := y.(UnitVal); ok {
			return UnitVal{V: float64(u) * v.V, F: v.F, T: v.T}, nil
		}
	case DoubleVal:
		if v, ok := y.(DoubleVal); ok {
			return u * v, nil
		}
		if v, ok := y.(UnitVal); ok {
			return UnitVal{V: float64(u) * v.V, F: v.F, T: v.T}, nil
		}
	case UnitVal:
		if v, ok := y.(IntVal); ok {
			return UnitVal{V: u.V * float64(v), F: u.F, T: u.T}, nil
		}
		if v, ok := y.(DoubleVal); ok {
			return UnitVal{V: u.V * float64(v), F: u.F, T: u.T}, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for *: %T and %T", x, y)
}
func div(x, y Val) (Val, error) {
	switch u := x.(type) {
	case IntVal:
		if v, ok := y.(IntVal); ok {
			return u / v, nil
		}
	case DoubleVal:
		if v, ok := y.(DoubleVal); ok {
			return u / v, nil
		}
	case UnitVal:
		if v, ok := y.(IntVal); ok {
			return UnitVal{V: u.V / float64(v), F: u.F, T: u.T}, nil
		}
		if v, ok := y.(DoubleVal); ok {
			return UnitVal{V: u.V / float64(v), F: u.F, T: u.T}, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for /: %T and %T", x, y)
}
func modulo(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return u % v, nil
		}
	}
	return nil, fmt.Errorf("incompatible types for %%: %T and %T", x, y)
}
func logicalAnd(x, y Val) (Val, error) {
	return BoolVal(x.Bool() && y.Bool()), nil
}
func logicalOr(x, y Val) (Val, error) {
	return BoolVal(x.Bool() || y.Bool()), nil
}

// Val equality is delegated to Go equality.
// This works as expected for scalar types.
// Records never compare equal.
func equal(x, y Val) (Val, error) {
	return BoolVal(x == y), nil
}
func notEqual(x, y Val) (Val, error) {
	return BoolVal(x != y), nil
}
func lessThan(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return BoolVal(u < v), nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return BoolVal(u < v), nil
		}
	} else if u, ok := x.(UnitVal); ok {
		if v, ok := y.(UnitVal); ok {
			if u.T == v.T {
				return BoolVal(unitCompare(u, v) < 0), nil
			}
		}
	}
	return nil, fmt.Errorf("incompatible types for <: %T and %T", x, y)
}
func lessEq(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return BoolVal(u <= v), nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return BoolVal(u <= v), nil
		}
	} else if u, ok := x.(UnitVal); ok {
		if v, ok := y.(UnitVal); ok {
			if u.T == v.T {
				return BoolVal(unitCompare(u, v) <= 0), nil
			}
		}
	}
	return nil, fmt.Errorf("incompatible types for <=: %T and %T", x, y)
}
func greaterThan(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return BoolVal(u > v), nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return BoolVal(u > v), nil
		}
	} else if u, ok := x.(UnitVal); ok {
		if v, ok := y.(UnitVal); ok {
			if u.T == v.T {
				return BoolVal(unitCompare(u, v) > 0), nil
			}
		}
	}
	return nil, fmt.Errorf("incompatible types for >: %T and %T", x, y)
}
func greaterEq(x, y Val) (Val, error) {
	if u, ok := x.(IntVal); ok {
		if v, ok := y.(IntVal); ok {
			return BoolVal(u >= v), nil
		}
	} else if u, ok := x.(DoubleVal); ok {
		if v, ok := y.(DoubleVal); ok {
			return BoolVal(u >= v), nil
		}
	} else if u, ok := x.(UnitVal); ok {
		if v, ok := y.(UnitVal); ok {
			if u.T == v.T {
				return BoolVal(unitCompare(u, v) >= 0), nil
			}
		}
	}
	return nil, fmt.Errorf("incompatible types for >=: %T and %T", x, y)
}

func unitCompare(u, v UnitVal) int {
	if u.T != v.T {
		log.Fatalf("unitCompare: arguments must be of same type, got %s and %s", u.TypeId(), v.TypeId())
	}
	x, y := u.V, v.V
	if u.F < v.F {
		y = v.V * (v.F / u.F)
	} else if u.F > v.F {
		x = u.V * (u.F / v.F)
	}
	if x < y {
		return -1
	} else if x == y {
		return 0
	}
	return 1
}

// Unary operations on Val.
func unaryMinus(x Val) (Val, error) {
	switch u := x.(type) {
	case IntVal:
		return -u, nil
	case DoubleVal:
		return -u, nil
	case UnitVal:
		return UnitVal{V: -u.V, F: u.F, T: u.T}, nil
	}
	return nil, fmt.Errorf("incompatible type for unary -: %T", x)
}

func unaryNot(x Val) (Val, error) {
	return BoolVal(!x.Bool()), nil
}

func unaryOp(x Val, op token.TokenType) (Val, error) {
	switch op {
	case token.Minus:
		return unaryMinus(x)
	case token.Not:
		return unaryNot(x)
	}
	return nil, fmt.Errorf("invalid unary operator '%v'", op)
}

func binaryOp(x, y Val, op token.TokenType) (Val, error) {
	switch op {
	case token.Plus:
		return plus(x, y)
	case token.Minus:
		return minus(x, y)
	case token.Times:
		return times(x, y)
	case token.Div:
		return div(x, y)
	case token.Modulo:
		return modulo(x, y)
	case token.LogicalAnd:
		return logicalAnd(x, y)
	case token.LogicalOr:
		return logicalOr(x, y)
	case token.Equal:
		return equal(x, y)
	case token.NotEqual:
		return notEqual(x, y)
	case token.LessThan:
		return lessThan(x, y)
	case token.LessEq:
		return lessEq(x, y)
	case token.GreaterThan:
		return greaterThan(x, y)
	case token.GreaterEq:
		return greaterEq(x, y)
	case token.Merge:
		return mergeValues(x, y)
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
		r, err := unaryOp(x, e.Op)
		if err != nil {
			return nil, &EvalError{pos: e.OpPos, msg: err.Error()}
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
		r, err := binaryOp(x, y, e.Op)
		if err != nil {
			return nil, &EvalError{pos: e.OpPos, msg: err.Error()}
		}
		return r, nil
	case *VarExpr:
		lval, vctx := ctx.Lookup(e.Name)
		if lval == nil {
			return nil, &EvalError{pos: e.Pos(), msg: fmt.Sprintf("unbound variable %s", e.Name)}
		}
		switch lv := lval.(type) {
		case *fullyEvaluated:
			return lv.val, nil
		case *lazyExpr:
			if vctx.isActive(e.Name) {
				return nil, &EvalError{pos: e.Pos(), msg: "cyclic variable dependencies detected"}
			}
			vctx.setActive(e.Name)
			v, err := Eval(lv.expr, vctx)
			if err != nil {
				return nil, err
			}
			vctx.store(e.Name, v)
			return v, nil
		default:
			log.Fatalf("Unhandled type for LazyVal: %T", lval)
		}
	case *RecExpr:
		rctx := ChildCtx(ctx)
		// Prepare context by storing lazy expressions of all fields.
		for _, lv := range e.LetVars {
			rctx.storeExpr(lv.Name, lv.X)
		}
		for _, f := range e.Fields {
			rctx.storeExpr(f.Name, f.X)
		}
		// Evaluate all let vars and fields.
		for _, lv := range e.LetVars {
			if _, found := rctx.fullyEvaluated(lv.Name); found {
				continue
			}
			rctx.setActive(lv.Name)
			v, err := Eval(lv.X, rctx)
			if err != nil {
				return nil, err
			}
			rctx.store(lv.Name, v)
		}
		rec := NewRec()
		for _, f := range e.Fields {
			var t *Typ = nil
			if f.T != nil {
				t = rctx.LookupType(f.T.TypeId())
				if t == nil {
					return nil, &EvalError{pos: f.T.Pos(), msg: fmt.Sprintf("unknown type %s for field %s", f.T.TypeId(), f.Name)}
				}
			}
			if v, found := rctx.fullyEvaluated(f.Name); found {
				// Eval of some other field already required evaluation of this field.
				if err := typeCheck(v.val, t); err != nil {
					return nil, &EvalError{pos: f.T.Pos(), msg: fmt.Sprintf("type error for field %s: %s", f.Name, err)}
				}
				rec.setField(f.Name, v.val, t)
				continue
			}
			rctx.setActive(f.Name)
			v, err := Eval(f.X, rctx)
			if err != nil {
				return nil, err
			}
			if t != nil {
				if err := typeCheck(v, t); err != nil {
					return nil, &EvalError{pos: f.T.Pos(), msg: fmt.Sprintf("type error for field %s: %s", f.Name, err)}
				}
				if u, ok := v.(UnitVal); ok {
					v = conformUnits(u, t, f.T.TypeId())
				}
			}
			rctx.store(f.Name, v)
			rec.setField(f.Name, v, t)
		}
		return rec, nil
	case *ListExpr:
		xs := make([]Val, len(e.Elements))
		for i, elem := range e.Elements {
			x, err := Eval(elem, ctx)
			if err != nil {
				return nil, err
			}
			xs[i] = x
		}
		return &ListVal{Elements: xs}, nil
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
			return nil, &EvalError{pos: e.End(), msg: fmt.Sprintf("record has no field '%s'", e.Name)}
		default:
			return nil, &EvalError{pos: e.End(), msg: fmt.Sprintf("cannot access .%s on type %T", e.Name, e)}
		}
	case *CallExpr:
		fe, err := Eval(e.Func, ctx)
		if err != nil {
			return nil, err
		}
		f, ok := fe.(CallableVal)
		if !ok {
			return nil, &EvalError{pos: e.Func.Pos(), msg: fmt.Sprintf("type %T is not callable", fe)}
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
	case *TypedExpr:
		val, err := Eval(e.X, ctx)
		if err != nil {
			return nil, err
		}
		return convertType(val, e.T.TypeId(), ctx, e.Pos())
	}
	return nil, &EvalError{pos: expr.Pos(), msg: fmt.Sprintf("not implemented: %T", expr)}
}

func mergeValues(x, y Val) (Val, error) {
	u, ok := x.(*RecVal)
	if !ok {
		return nil, fmt.Errorf("cannot merge lhs of type %T", x)
	}
	v, ok := y.(*RecVal)
	if !ok {
		return nil, fmt.Errorf("cannot merge rhs of type %T", y)
	}
	r := NewRec()
	if err := mergeRecVal(u, v, r); err != nil {
		return nil, err
	}
	return r, nil
}

func mergeRecVal(x, y, r *RecVal) error {
	// Copy fields only in x.
	for f, vx := range x.Fields {
		if _, ok := y.Fields[f]; !ok {
			r.setField(f, vx, x.FieldTypes[f])
		}
	}
	// Copy fields only in y and merge common fields.
	for f, vy := range y.Fields {
		if vx, ok := x.Fields[f]; !ok {
			// Unique field of y.
			r.setField(f, vy, y.FieldTypes[f])
		} else {
			// Common field.
			// If only x has a type annotation, only allow merging if y's value has the same type
			// OR y has an explicit type annotation (i.e. interpret y's annotation as an explicit override).
			tx, xHasType := x.FieldTypes[f]
			ty, yHasType := y.FieldTypes[f]
			if xHasType && !yHasType {
				if err := typeCheck(vy, tx); err != nil {
					return err
				}
			}
			targetType := tx
			if yHasType {
				targetType = ty
			}
			// TODO: TypedVal is not properly supported here: we wouldn't recurse into a typed rec.
			if _, ok := vx.(*TypedVal); ok {
				return fmt.Errorf("merging TypedVal is not implemented")
			}
			if _, ok := vy.(*TypedVal); ok {
				return fmt.Errorf("merging TypedVal is not implemented")
			}
			if ry, ok := vy.(*RecVal); !ok {
				// y field is not a record, just take the value from y.
				r.setField(f, vy, targetType)
			} else if rx, ok := vx.(*RecVal); ok {
				// x's field is a record, too: recurse.
				cr := NewRec()
				r.setField(f, cr, targetType)
				mergeRecVal(rx, ry, cr)
			} else {
				// x field is not a record, again just take the value from y.
				r.setField(f, vy, targetType)
			}
		}
	}
	return nil
}
