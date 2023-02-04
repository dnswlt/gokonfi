package gokonfi

import (
	"fmt"
	"strings"
)

// Declaration of all built-in functions. Whatever we add here
// becomes available in Konfi.
var builtinFunctions = []*NativeFuncVal{
	{Name: "len", Arity: 1, F: builtinLen},
	{Name: "contains", Arity: 2, F: builtinContains},
	{Name: "cond", Arity: 3, F: builtinCond},
	{Name: "format", Arity: -1, F: builtinFormat},
	{Name: "str", Arity: 1, F: builtinStr},
	{Name: "isnil", Arity: 1, F: builtinIsnil},
	{Name: "flatmap", Arity: 2, F: builtinFlatmap},
	{Name: "fold", Arity: -1, F: builtinFold},
}

func builtinLen(args []Val, ctx *Ctx) (Val, error) {
	switch arg := args[0].(type) {
	case StringVal:
		return IntVal(len(arg)), nil
	case *RecVal:
		return IntVal(len(arg.Fields)), nil
	case *ListVal:
		return IntVal(len(arg.Elements)), nil
	}
	return nil, fmt.Errorf("len: invalid type: %T", args[0])
}

func builtinContains(args []Val, ctx *Ctx) (Val, error) {
	switch s := args[0].(type) {
	case StringVal:
		if substr, ok := args[1].(StringVal); ok {
			return BoolVal(strings.Contains(string(s), string(substr))), nil
		}
		return nil, fmt.Errorf("contains: invalid type for second argument: %T", args[1])
	}
	return nil, fmt.Errorf("contains: invalid argument types: (%T, %T)", args[0], args[1])
}

func builtinCond(args []Val, ctx *Ctx) (Val, error) {
	if args[0].Bool() {
		return args[1], nil
	}
	return args[2], nil
}

func builtinFormat(args []Val, ctx *Ctx) (Val, error) {
	if len(args) == 0 {
		return StringVal(""), nil
	}
	if len(args) == 1 {
		return args[1], nil
	}
	format, ok := args[0].(StringVal)
	if !ok {
		return nil, fmt.Errorf("format: first argument must be a format string, got %T", args[0])
	}
	formatArgs := make([]any, len(args[1:]))
	for i, arg := range args[1:] {
		formatArgs[i] = arg
	}
	s := fmt.Sprintf(string(format), formatArgs...)
	return StringVal(s), nil
}

func builtinStr(args []Val, ctx *Ctx) (Val, error) {
	return StringVal(args[0].String()), nil
}

func builtinIsnil(args []Val, ctx *Ctx) (Val, error) {
	_, ok := args[0].(NilVal)
	return BoolVal(ok), nil
}

func builtinFlatmap(args []Val, ctx *Ctx) (Val, error) {
	f, ok := args[0].(CallableVal)
	if !ok {
		return nil, fmt.Errorf("flatmap: 1st argument must be a callable, got %T", args[1])
	}
	xs, ok := args[1].(*ListVal)
	if !ok {
		return nil, fmt.Errorf("flatmap: 2nd argument must be a list, got %T", args[0])
	}
	result := []Val{}
	for _, x := range xs.Elements {
		fx, err := f.Call([]Val{x}, ctx)
		if err != nil {
			return nil, fmt.Errorf("flatmap: call failed: %w", err)
		}
		if ys, ok := fx.(*ListVal); ok {
			// f returned a list (as it should): append all elements to the result.
			result = append(result, ys.Elements...)
		} else {
			// Value returned by f was not a list: append single value.
			result = append(result, ys)
		}
	}
	return &ListVal{Elements: result}, nil
}

func builtinFold(args []Val, ctx *Ctx) (Val, error) {
	if len(args) != 3 && len(args) != 2 {
		return nil, fmt.Errorf("fold: invalid number of arguments: %d", len(args))
	}
	if len(args) == 2 {
		return builtinFold1(args, ctx)
	}
	f, ok := args[0].(CallableVal)
	if !ok {
		return nil, fmt.Errorf("fold: 1st argument must be a callable, got %T", args[1])
	}
	xs, ok := args[2].(*ListVal)
	if !ok {
		return nil, fmt.Errorf("fold: 3nd argument must be a list, got %T", args[0])
	}
	accu := args[1]
	for _, x := range xs.Elements {
		y, err := f.Call([]Val{accu, x}, ctx)
		if err != nil {
			return nil, fmt.Errorf("fold: call failed: %w", err)
		}
		accu = y
	}
	return accu, nil
}

func builtinFold1(args []Val, ctx *Ctx) (Val, error) {
	f, ok := args[0].(CallableVal)
	if !ok {
		return nil, fmt.Errorf("fold: 1st argument must be a callable, got %T", args[1])
	}
	xs, ok := args[1].(*ListVal)
	if !ok {
		return nil, fmt.Errorf("fold: 2nd argument must be a list, got %T", args[0])
	}
	if len(xs.Elements) == 0 {
		return NilVal{}, nil
	}
	accu := xs.Elements[0]
	for _, x := range xs.Elements[1:] {
		y, err := f.Call([]Val{accu, x}, ctx)
		if err != nil {
			return nil, fmt.Errorf("fold: call failed: %w", err)
		}
		accu = y
	}
	return accu, nil
}
