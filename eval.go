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

// A lazyVal is a wrapper for "lazy" values, which can be one of
// - a fully evaluated Val
// - an Expr that still needs to be evaluated
type lazyVal struct {
	expr Expr
	val  Val
}

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
	name    string         // Name of this module. In practice, always its file path.
	pubVars map[string]Val // Declared pub(lic) variables of the module.
	body    Val            // The final (optional) module body. Set to NilVal{} if not present.
}

func (m *loadedModule) Body() Val {
	return m.body
}

func (m *loadedModule) AsRec() *RecVal {
	r := NewRec()
	for v, val := range m.pubVars {
		r.setField(v, val, nil) // Module-level vars have no FieldAnnotation.
	}
	const bodyField = "body"
	if _, ok := r.Fields[bodyField]; !ok {
		// A module-level declaration can hide the body and make it inaccessible.
		// This is not terribly beautiful, but works well enough in practice.
		r.setField(bodyField, m.body, nil)
	}
	return r
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

// ChildCtx creates a child context of the given parent.
// It shares the global state with its parent. Changes to the global state
// will be visible in both.
func ChildCtx(parent *Ctx) *Ctx {
	return &Ctx{
		vars: &varCtx{
			env:    make(map[string]lazyVal),
			active: make(map[string]bool),
			parent: parent.vars},
		global: parent.global,
	}
}

// GlobalCtx returns a context that contains all builtin functions and types.
func GlobalCtx() *Ctx {
	ctx := EmptyCtx()
	for _, builtin := range builtinFunctions {
		ctx.store(builtin.Name, builtin)
	}
	for _, typ := range builtinTypes {
		ctx.defineType(typ)
	}
	return ctx
}

// Returns the top-level context of ctx. This context typically contains only the
// builtin functions. It shares the global state with ctx and should be used when
// loading a module from another module.
func (ctx *Ctx) dropLocals() *Ctx {
	// The last varCtx in the chain contains the global variables.
	l := ctx.vars
	for l.parent != nil {
		l = l.parent
	}
	return &Ctx{global: ctx.global, vars: l}
}

// Looks up the value of v in ctx. It also returns the (parent) context
// in which v is defined. If no definition for v exists, it returns an empty
// lazyVal and a nil context.
func (ctx *Ctx) Lookup(v string) (lazyVal, *Ctx) {
	c := ctx.vars
	for c != nil {
		if val, ok := c.env[v]; ok {
			return val, &Ctx{c, ctx.global}
		}
		c = c.parent
	}
	return lazyVal{}, nil // Not found
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
func (ctx *Ctx) fullyEvaluated(v string) (val Val, found bool) {
	if lv, ok := ctx.vars.env[v]; ok {
		if lv.val != nil {
			return lv.val, true
		}
	}
	return nil, false
}

// Stores the given value in ctx under name v. Also removes v from the set of active variables.
func (ctx *Ctx) store(v string, val Val) {
	ctx.vars.env[v] = lazyVal{val: val}
	delete(ctx.vars.active, v)
}

func (ctx *Ctx) storeExpr(v string, expr Expr) {
	ctx.vars.env[v] = lazyVal{expr: expr}
}

func (ctx *Ctx) storeModule(m *loadedModule) {
	ctx.global.modules[m.name] = m
}

func (ctx *Ctx) defineType(typ *Typ) {
	ctx.global.types[typ.Id] = typ
	for n := range typ.UnitMults {
		ctx.global.types[n] = typ
	}
}

func (ctx *Ctx) addFile(name string, size int) *token.File {
	return ctx.global.fileset.AddFile(name, size)
}

// isActiveFile checks if a file with the given name is currently on the
// evaluation stack. This is used to detect import cycles.
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

// cwd returns the current working directory of ctx. If the stack is not empty,
// this is always the directory of the file on top of the stack. Otherwise, it is ".".
func (ctx *Ctx) cwd() string {
	if len(ctx.global.filestack) == 0 {
		return "."
	}
	return path.Dir(ctx.global.filestack[len(ctx.global.filestack)-1])
}

func (ctx *Ctx) FileSet() *token.FileSet {
	return ctx.global.fileset
}

// EvalError is the error type commonly returned if evaluation of an expression or module fails.
type EvalError struct {
	pos   token.Pos // Position at which evaluation failed.
	msg   string    // Error message.
	cause error     // Optional root cause error.
}

func (e *EvalError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("EvalError: %s (caused by: %s) at position %d", e.msg, e.cause, e.pos)
	}
	return fmt.Sprintf("EvalError: %s at position %d", e.msg, e.pos)
}

func (e *EvalError) Pos() token.Pos {
	return e.pos
}

func (e *EvalError) Unwrap() error {
	return e.cause
}

// RecVal represents record values, a.k.a. dicts, structs, objects.
type RecVal struct {
	Fields           map[string]Val
	FieldAnnotations map[string]*FieldAnnotation // Optional type annotations per field.
}

// Information about the type annotation attached to a record field,
// e.g. the minutes in `{ x::minutes }`.
type FieldAnnotation struct {
	T *Typ
	M float64 // optional, only nonzero for unit types (for which T.IsUnit() is true).
}

// NewRec returns a new record with no fields.
func NewRec() *RecVal {
	return &RecVal{Fields: make(map[string]Val), FieldAnnotations: make(map[string]*FieldAnnotation)}
}

func NewRecWithFields(fields map[string]Val) *RecVal {
	return &RecVal{Fields: fields, FieldAnnotations: make(map[string]*FieldAnnotation)}
}

func (r *RecVal) setField(field string, val Val, anno *FieldAnnotation) {
	r.Fields[field] = val
	if anno != nil {
		r.FieldAnnotations[field] = anno
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

func (u UnitVal) TypeId() string {
	return u.T.Id
}

func (u UnitVal) WithF(f float64) UnitVal {
	if u.F == f {
		return u
	}
	return UnitVal{V: u.V * (u.F / f), F: f, T: u.T}
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

type TypedVal struct {
	V Val
	T *Typ
}

func (v *TypedVal) TypeId() string {
	return v.T.Id
}

func (v IntVal) valImpl()         {}
func (v DoubleVal) valImpl()      {}
func (v UnitVal) valImpl()        {}
func (v BoolVal) valImpl()        {}
func (v StringVal) valImpl()      {}
func (v NilVal) valImpl()         {}
func (v *RecVal) valImpl()        {}
func (v ListVal) valImpl()        {}
func (v *NativeFuncVal) valImpl() {}
func (v *FuncExprVal) valImpl()   {}
func (v TypedVal) valImpl()       {}

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
func (r ListVal) Bool() bool {
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
	if n, ok := v.T.unitMultiplierName(v.F); ok {
		f := strconv.FormatFloat(v.V, 'f', -1, 64)
		return f + "::" + n
	}
	// A UnitVal with an unknown unit is an interpreter bug.
	panic(fmt.Sprintf("UnitVal %s with invalid factor %f", v.TypeId(), v.F))
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
func (r ListVal) String() string {
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
func (r ListVal) Typ() *Typ {
	return builtinTypeList
}
func (r *NativeFuncVal) Typ() *Typ {
	return builtinTypeNativeFunc
}
func (r *FuncExprVal) Typ() *Typ {
	return builtinTypeFuncExpr
}
func (v TypedVal) Typ() *Typ {
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
		lv, vctx := ctx.Lookup(e.Name)
		if vctx == nil {
			return nil, &EvalError{pos: e.Pos(), msg: fmt.Sprintf("unbound variable %s", e.Name)}
		}
		switch {
		case lv.val != nil:
			return lv.val, nil
		case lv.expr != nil:
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
			log.Fatalf("lazyVal with nil .val and .expr for variable %s", e.Name)
		}
	case *RecExpr:
		return evalRec(e, ctx)
	case *ListExpr:
		xs := make([]Val, len(e.Elements))
		for i, elem := range e.Elements {
			x, err := Eval(elem, ctx)
			if err != nil {
				return nil, err
			}
			xs[i] = x
		}
		return ListVal{Elements: xs}, nil
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
			return nil, &EvalError{pos: e.DotPos, msg: fmt.Sprintf("record has no field '%s'", e.Name)}
		case TypedVal:
			if rv, ok := r.V.(*RecVal); ok {
				if v, ok := rv.Fields[e.Name]; ok {
					return v, nil
				}
			}
			return nil, &EvalError{pos: e.End(), msg: fmt.Sprintf("%s has no field '%s'", r.Typ().Id, e.Name)}
		default:
			return nil, &EvalError{pos: e.End(), msg: fmt.Sprintf("cannot access .%s on type %s", e.Name, r.Typ().Id)}
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
		if err != nil {
			return nil, &EvalError{pos: e.Func.Pos(), msg: "call failed", cause: err}
		}
		return res, nil
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
	return nil, &EvalError{pos: expr.Pos(), msg: fmt.Sprintf("Eval: not implemented: %T", expr)}
}

func evalRec(e *RecExpr, ctx *Ctx) (Val, error) {
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
		var t *Typ
		m := 0.
		if f.T != nil {
			t = rctx.LookupType(f.T.TypeId())
			if t == nil {
				return nil, &EvalError{pos: f.T.Pos(), msg: fmt.Sprintf("unknown type %s for field %s", f.T.TypeId(), f.Name)}
			}
			if t.IsUnit() {
				// f.T may be the unit type itself (allowing any multiplier),
				// so UnitMults may return 0 here.
				m = t.UnitMults[f.T.TypeId()]
			}
		}
		var v Val
		cv, found := rctx.fullyEvaluated(f.Name)
		if found {
			// Eval of another expression already required evaluation of this field.
			v = cv
		} else {
			var err error
			rctx.setActive(f.Name)
			v, err = Eval(f.X, rctx)
			if err != nil {
				return nil, err
			}
			rctx.store(f.Name, v)
		}
		if t != nil {
			// Typed field
			if err := typeCheck(v, t); err != nil {
				return nil, &EvalError{pos: f.T.Pos(), msg: fmt.Sprintf("type error for field %s: %s", f.Name, err)}
			}
			if u, ok := v.(UnitVal); ok && m > 0. {
				v = u.WithF(m)
			}
			rec.setField(f.Name, v, &FieldAnnotation{T: t, M: m})
		} else {
			// t == nil => Untyped field
			rec.setField(f.Name, v, nil)
		}
	}
	return rec, nil
}

// Evaluates the given module m.
// If the module has type or unit declarations, those will be added to ctx.
func EvalModule(m *Module, ctx *Ctx) (*loadedModule, error) {
	mctx := ChildCtx(ctx)
	for _, d := range m.LetVars {
		mctx.storeExpr(d.Name, d.X)
	}
	for _, d := range m.PubDecls {
		mctx.storeExpr(d.Name, d.X)
	}
	// Evaluate type declarations first. Types declared in a module can be used by
	// expressions, pub declarations and let bindings in that module. But the opposite
	// is not true: types declarations can only use what's already defined before the
	// module is evaluated and those declarations of the module that don't depend on the
	// type being declared.
	for _, d := range m.UnitDecls {
		val, err := Eval(d.Multiples, mctx)
		if err != nil {
			return nil, err
		}
		rv, ok := val.(*RecVal)
		if !ok {
			panic(fmt.Sprintf("*RecExpr must evaluate to *RecVal, got %s", rv.Typ().Id))
		}
		// Collect multiples
		unitMults := make(map[string]float64)
		for f, v := range rv.Fields {
			// Can be either int or double, for convenience.
			switch u := v.(type) {
			case IntVal:
				unitMults[f] = float64(u)
			case DoubleVal:
				unitMults[f] = float64(u)
			default:
				return nil, &EvalError{pos: d.Multiples.Fields[f].X.Pos(), msg: fmt.Sprintf("Invalid type for multiplier %s: %s", f, v.Typ().Id)}
			}
		}
		t := NewUnitType(d.Name, unitMults)
		ctx.defineType(t)
	}
	// Evaluate module-level declarations. This is mostly analogous to how records are evaluated.
	for _, d := range m.LetVars {
		if _, found := mctx.fullyEvaluated(d.Name); found {
			continue
		}
		mctx.setActive(d.Name)
		v, err := Eval(d.X, mctx)
		if err != nil {
			return nil, err
		}
		mctx.store(d.Name, v)
	}
	pubVars := make(map[string]Val)
	for _, d := range m.PubDecls {
		if v, found := mctx.fullyEvaluated(d.Name); found {
			pubVars[d.Name] = v
			continue
		}
		mctx.setActive(d.Name)
		v, err := Eval(d.X, mctx)
		if err != nil {
			return nil, err
		}
		mctx.store(d.Name, v)
		pubVars[d.Name] = v
	}
	// Evaluate body in a context that is aware of all declarations.
	var body Val = NilVal{}
	if m.Body != nil {
		v, err := Eval(m.Body, mctx)
		if err != nil {
			return nil, err
		}
		body = v
	}
	return &loadedModule{name: m.Name, pubVars: pubVars, body: body}, nil
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
			r.setField(f, vx, x.FieldAnnotations[f])
		}
	}
	// Copy fields only in y and merge common fields.
	for f, vy := range y.Fields {
		if vx, ok := x.Fields[f]; !ok {
			// Unique field of y.
			r.setField(f, vy, y.FieldAnnotations[f])
		} else {
			// Common field.
			// If only x has a type annotation, only allow merging if y's value has the same type
			// OR y has an explicit type annotation (i.e. interpret y's annotation as an explicit override).
			ax, xHasType := x.FieldAnnotations[f]
			ay, yHasType := y.FieldAnnotations[f]
			if xHasType && !yHasType {
				if err := typeCheck(vy, ax.T); err != nil {
					return fmt.Errorf("type error merging record field '%s': %w", f, err)
				}
				if ax.T.IsUnit() {
					if uy, ok := vy.(UnitVal); ok {
						if ax.M > 0 {
							vy = uy.WithF(ax.M)
						}
					} else {
						// Implementation error if we end up here: if vy passes the type check for tx,
						// it must be a UnitVal.
						log.Fatalf("%v passes type check for type %s but is not a unit", vy, ax.T.Id)
					}
				}
			}
			targetType := ax
			if yHasType {
				targetType = ay
			}
			// TODO: TypedVal is not properly supported here: we wouldn't recurse into a typed rec.
			if _, ok := vx.(TypedVal); ok {
				return fmt.Errorf("merging TypedVal is not implemented")
			}
			if _, ok := vy.(TypedVal); ok {
				return fmt.Errorf("merging TypedVal is not implemented")
			}
			if ry, ok := vy.(*RecVal); ok {
				if rx, ok := vx.(*RecVal); ok {
					// x and y are records: recurse
					cr := NewRec()
					r.setField(f, cr, targetType)
					if err := mergeRecVal(rx, ry, cr); err != nil {
						return err
					}
					continue
				}
			}
			// Just take the value from y.
			r.setField(f, vy, targetType)
		}
	}
	return nil
}
