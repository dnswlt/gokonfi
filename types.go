package gokonfi

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/dnswlt/gokonfi/token"
)

type UnitType struct {
	Name   string
	Factor float64
}

type Typ struct {
	Id       string
	Convert  CallableVal // (any) -> Self
	Unwrap   CallableVal // (Self) -> Val
	Validate CallableVal // (Self) -> bool
	Units    []UnitType  // Non-nil for unit types. Sorted by Name.
}

func (t *Typ) IsUnit() bool {
	return len(t.Units) > 0
}

func (t *Typ) UnitFactor(unit string) (v float64, found bool) {
	if i, found := sort.Find(len(t.Units), func(i int) int {
		return strings.Compare(unit, t.Units[i].Name)
	}); found {
		return t.Units[i].Factor, true
	}
	return 1., false
}

func (t *Typ) UnitName(factor float64) (name string, found bool) {
	for _, u := range t.Units {
		if u.Factor == factor {
			return u.Name, true
		}
	}
	return "", false
}

var (
	// Predefine built-in types, so we can use == comparisons for those.
	builtinTypeBool     = &Typ{Id: "bool"}
	builtinTypeInt      = &Typ{Id: "int"}
	builtinTypeDouble   = &Typ{Id: "double"}
	builtinTypeString   = &Typ{Id: "string"}
	builtinTypeDuration = &Typ{
		Id: "duration",
		Convert: &NativeFuncVal{
			Name:  "duration.Convert",
			F:     nil, // initialized in init() to avoid a cycle.
			Arity: 2,
		},
		Unwrap: &NativeFuncVal{
			Name: "duration.Unwrap",
			F: func(args []Val, ctx *Ctx) (Val, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("duration.Unwrap: want 1 argument, got %d", len(args))
				}
				uval, ok := args[0].(*UnitVal)
				if !ok {
					return nil, fmt.Errorf("duration.Unwrap: want TypedVal argument, got %T", args[0])
				}
				if uval.T.Id != "duration" {
					return nil, fmt.Errorf("duration.Unwrap: called on invalid type: %s", uval.TypeId())
				}
				return DoubleVal(uval.V), nil
			},
			Arity: 1,
		},
		Units: []UnitType{
			{"nanos", 1},
			{"micros", 1000},
			{"millis", 1000 * 1000},
			{"seconds", 1000 * 1000 * 1000},
			{"minutes", 1000 * 1000 * 1000 * 60},
			{"hours", 1000 * 1000 * 1000 * 60 * 60},
			{"days", 1000 * 1000 * 1000 * 60 * 60 * 24},
		}}

	// Gets further extended in the init function.
	builtinTypes = []*Typ{
		builtinTypeBool,
		builtinTypeInt,
		builtinTypeDouble,
		builtinTypeString,
		builtinTypeDuration,
	}
)

func init() {
	// Initialize Convert function(s) here to avoid a cyclic dependency.
	builtinTypeDuration.Convert.(*NativeFuncVal).F = func(args []Val, ctx *Ctx) (Val, error) {
		return builtinUnitTypeConvert(builtinTypeDuration, args, ctx)
	}
	// TODO: this sorting requirement is a bit silly.
	sort.Slice(builtinTypeDuration.Units, func(i, j int) bool {
		return builtinTypeDuration.Units[i].Name < builtinTypeDuration.Units[j].Name
	})
}

func builtinUnitTypeConvert(typ *Typ, args []Val, ctx *Ctx) (Val, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s.Convert: want 2 arguments, got %d", typ.Id, len(args))
	}
	unit, ok := args[0].(StringVal)
	if !ok {
		return nil, fmt.Errorf("%s.Convert: want 1st argument as StringVal, got %T", typ.Id, args[0])
	}
	f, ok := typ.UnitFactor(string(unit))
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
	switch v := val.(type) {
	case IntVal:
		if t == builtinTypeInt {
			return nil
		}
	case DoubleVal:
		if t == builtinTypeDouble {
			return nil
		}
	case StringVal:
		if t == builtinTypeString {
			return nil
		}
	case UnitVal:
		if v.T == t {
			return nil
		}
	}
	return fmt.Errorf("incompatible types: %T :: %s", val, t.Id)
}

func conformUnits(u UnitVal, t *Typ, target string) UnitVal {
	if u.T != t {
		log.Fatalf("Cannot conform units of different types: %s <> %s", u.T.Id, t.Id)
	}
	f, found := t.UnitFactor(target)
	if !found {
		log.Fatalf("Invalid unit multiplier name: %s", target)
	}
	if u.F == f {
		return u
	}
	return UnitVal{V: u.V * (u.F / f), F: f, T: t}
}
