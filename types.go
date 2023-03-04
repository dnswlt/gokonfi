package gokonfi

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dnswlt/gokonfi/token"
)

type Typ struct {
	Id        string
	Convert   CallableVal        // (string, any) -> Self
	Encode    CallableVal        // (Self) -> Val
	Validate  CallableVal        // (Self) -> bool
	UnitMults map[string]float64 // Non-nil only for unit types.
}

func (t *Typ) IsUnit() bool {
	return len(t.UnitMults) > 0
}

func (t *Typ) unitMultiplierName(factor float64) (name string, found bool) {
	for n, f := range t.UnitMults {
		if f == factor {
			return n, true
		}
	}
	return "", false
}

// NewUnitType returns a new unit type. Callers must populate its UnitMults afterwards.
func NewUnitType(name string, unitMults map[string]float64) *Typ {
	t := &Typ{
		Id: name,
	}
	t.Convert = &NativeFuncVal{
		Name: name + ".Convert",
		F: func(args []Val, ctx *Ctx) (Val, error) {
			return builtinUnitTypeConvert(t, args, ctx)
		},
		Arity: 2,
	}
	t.Encode = &NativeFuncVal{
		Name: name + ".Encode",
		F: func(args []Val, ctx *Ctx) (Val, error) {
			return builtinUnitTypeEncode(t, args, ctx)
		},
		Arity: 1,
	}
	// Copy unitMults so callers don't accidentally modify them.
	um := make(map[string]float64)
	for k, v := range unitMults {
		um[k] = v
	}
	t.UnitMults = um
	return t
}

var (
	// Predefine built-in types. Type comparisons generally use pointer equality (==), so don't duplicate these types.
	builtinTypeBool       = &Typ{Id: "bool"}
	builtinTypeInt        = &Typ{Id: "int"}
	builtinTypeDouble     = &Typ{Id: "double"}
	builtinTypeString     = &Typ{Id: "string"}
	builtinTypeNil        = &Typ{Id: "nil"}
	builtinTypeRec        = &Typ{Id: "rec"}
	builtinTypeList       = &Typ{Id: "list"}
	builtinTypeNativeFunc = &Typ{Id: "builtin"}
	builtinTypeFuncExpr   = &Typ{Id: "func"}
	builtinTypeDuration   = NewUnitType("duration", map[string]float64{
		"nanos":   1,
		"micros":  1000,
		"millis":  1000 * 1000,
		"seconds": 1000 * 1000 * 1000,
		"minutes": 1000 * 1000 * 1000 * 60,
		"hours":   1000 * 1000 * 1000 * 60 * 60,
		"days":    1000 * 1000 * 1000 * 60 * 60 * 24,
	})
	builtinTypeTime = makeBuiltinTypeTime()

	// This slice contains all predefined (builtin) types. Add new types here to make them
	// available in konfi.
	builtinTypes = []*Typ{
		builtinTypeBool,
		builtinTypeInt,
		builtinTypeDouble,
		builtinTypeString,
		builtinTypeNil,
		builtinTypeRec,
		builtinTypeList,
		builtinTypeNativeFunc,
		builtinTypeFuncExpr,
		builtinTypeDuration,
		builtinTypeTime,
	}
)

func makeBuiltinTypeTime() *Typ {
	t := &Typ{Id: "time"}
	t.Convert = &NativeFuncVal{
		Name: "time.Convert",
		F: func(args []Val, ctx *Ctx) (Val, error) {
			switch a := args[1].(type) {
			case StringVal:
				tm, err := builtinLenientParseTime([]Val{a}, nil)
				if err != nil {
					return nil, err
				}
				return TypedVal{V: tm, T: t}, nil
			}
			return nil, fmt.Errorf("time.Convert: invalid argument type %s", args[0].Typ().Id)
		},
		Arity: 2,
	}
	t.Encode = &NativeFuncVal{
		Name: "time.Encode",
		F: func(args []Val, ctx *Ctx) (Val, error) {
			switch a := args[0].(type) {
			case TypedVal:
				if a.T != t {
					break
				}
				r := a.V.(*RecVal)
				intf := func(f string) int {
					i, ok := r.Fields[f].(IntVal)
					if !ok {
						return 0
					}
					return int(i)
				}
				loc := time.FixedZone("A51", intf("offset"))
				tm := time.Date(intf("year"), time.Month(intf("month")), intf("day"),
					intf("hour"), intf("minute"), intf("second"), intf("nanosecond"), loc)
				return StringVal(tm.Format("2006-01-02T15:04:05Z07:00")), nil
			}
			return nil, fmt.Errorf("time.Encode: invalid argument type %s", args[0].Typ().Id)
		},
		Arity: 1,
	}
	return t
}

func builtinUnitTypeConvert(typ *Typ, args []Val, _ *Ctx) (Val, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s.Convert: want 2 arguments, got %d", typ.Id, len(args))
	}
	unit, ok := args[0].(StringVal)
	if !ok {
		return nil, fmt.Errorf("%s.Convert: want 1st argument as StringVal, got %T", typ.Id, args[0])
	}
	f, ok := typ.UnitMults[string(unit)]
	if !ok {
		return nil, fmt.Errorf("%s.Convert: invalid unit %s", typ.Id, unit)
	}
	switch v := args[1].(type) {
	case DoubleVal:
		return UnitVal{V: float64(v), F: f, T: typ}, nil
	case IntVal:
		return UnitVal{V: float64(v), F: f, T: typ}, nil
	case UnitVal:
		if v.T == typ {
			if v.F == f {
				// same unit as before, nothing to do.
				return v, nil
			}
			return UnitVal{V: v.V * (v.F / f), F: f, T: v.T}, nil
		}
	}
	return nil, fmt.Errorf("%s.Convert: cannot convert from type %T", typ.Id, args[1])
}

func builtinUnitTypeEncode(typ *Typ, args []Val, _ *Ctx) (Val, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s.Encode: want 1 argument, got %d", typ.Id, len(args))
	}
	uval, ok := args[0].(UnitVal)
	if !ok {
		return nil, fmt.Errorf("%s.Encode: want UnitVal argument, got %T", typ.Id, args[0])
	}
	if uval.TypeId() != typ.Id {
		return nil, fmt.Errorf("%s.Encode: called on invalid type: %s", typ.Id, uval.TypeId())
	}
	return DoubleVal(uval.V), nil
}

func convertType(val Val, typeName string, ctx *Ctx, pos token.Pos) (Val, error) {
	typ := ctx.LookupType(typeName)
	if typ == nil {
		return nil, &EvalError{pos: pos, msg: fmt.Sprintf("unknown type: %s", typeName)}
	}
	if typ.Convert != nil {
		// Types with custom conversion functions convert themselves:
		return typ.Convert.Call([]Val{StringVal(typeName), val}, ctx)
	}
	// Everything can be converted to a bool.
	if typ == builtinTypeBool {
		return BoolVal(val.Bool()), nil
	}
	// Try other primitive types:
	switch v := val.(type) {
	case BoolVal:
		switch typ {
		case builtinTypeInt:
			i := 0
			if v {
				i = 1
			}
			return IntVal(i), nil
		case builtinTypeDouble:
			d := 0.
			if v {
				d = 1.
			}
			return DoubleVal(d), nil
		case builtinTypeString:
			return StringVal(v.String()), nil
		}
	case IntVal:
		switch typ {
		case builtinTypeInt:
			return val, nil
		case builtinTypeDouble:
			return DoubleVal(float64(v)), nil
		case builtinTypeString:
			return StringVal(v.String()), nil
		}
	case DoubleVal:
		switch typ {
		case builtinTypeInt:
			return IntVal(int64(v)), nil
		case builtinTypeDouble:
			return val, nil
		case builtinTypeString:
			return StringVal(v.String()), nil
		}
	case StringVal:
		switch typ {
		case builtinTypeInt:
			i, err := strconv.ParseInt(string(v), 10, 64)
			if err != nil {
				return nil, &EvalError{pos: pos, msg: fmt.Sprintf("cannot convert string %q to int", string(v))}
			}
			return IntVal(i), nil
		case builtinTypeDouble:
			d, err := strconv.ParseFloat(string(v), 64)
			if err != nil {
				return nil, &EvalError{pos: pos, msg: fmt.Sprintf("cannot convert string %q to double", string(v))}
			}
			return DoubleVal(d), nil
		case builtinTypeString:
			return val, nil
		}
	case UnitVal:
		// UnitVal is converted to int/double as its value in the current multiple,
		// unless the unit defines its own coversion function (like builtin ones do).
		switch typ {
		case builtinTypeInt:
			return IntVal(v.V), nil
		case builtinTypeDouble:
			return DoubleVal(v.V), nil
		}
	}
	return nil, &EvalError{pos: pos, msg: fmt.Sprintf("cannot convert value of type %T to %s", val, typ.Id)}
}

func typeCheck(val Val, t *Typ) error {
	if t == nil {
		// Type check against no type succeeds.
		return nil
	}
	if t == val.Typ() {
		return nil
	}
	return fmt.Errorf("incompatible types: %s <> %s", val.Typ().Id, t.Id)
}
