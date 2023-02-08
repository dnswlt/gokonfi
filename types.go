package gokonfi

import (
	"fmt"
	"strconv"

	"github.com/dnswlt/gokonfi/token"
)

type UnitType struct {
	Name   string
	Factor int
}

type Typ struct {
	Id       string
	Convert  CallableVal // (any) -> Self
	Unwrap   CallableVal // (Self) -> Val
	Validate CallableVal // (Self) -> bool
	Units    []UnitType
}

var (
	// Predefine built-in types, so we can use == comparisons for those.
	builtinTypeBool   = &Typ{Id: "bool"}
	builtinTypeInt    = &Typ{Id: "int"}
	builtinTypeDouble = &Typ{Id: "double"}
	builtinTypeString = &Typ{Id: "string"}

	builtinTypes = []*Typ{
		builtinTypeBool,
		builtinTypeInt,
		builtinTypeDouble,
		builtinTypeString,
	}
)

func convertType(val Val, typ *Typ, ctx *Ctx, pos token.Pos) (Val, error) {
	if typ.Convert != nil {
		// Types with custom conversion functions convert themselves:
		return typ.Convert.Call([]Val{val}, ctx)
	}
	// Everything can be converted to a bool.
	if typ == builtinTypeBool {
		return BoolVal(val.Bool()), nil
	}
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
	}
	return nil, &EvalError{pos: pos, msg: fmt.Sprintf("cannot convert value of type %T to %s", val, typ.Id)}
}
