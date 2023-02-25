package gokonfi

import (
	"fmt"
	"strings"
)

// Declaration of all built-in functions. Whatever we add here
// becomes available in Konfi.
//
// Keep sorted alphabetically.
var builtinFunctions = []*NativeFuncVal{
	{Name: "cond", Arity: 3, F: builtinCond},
	{Name: "contains", Arity: 2, F: builtinContains},
	{Name: "error", Arity: 1, F: builtinError},
	{Name: "flatmap", Arity: 2, F: builtinFlatmap},
	{Name: "fold", Arity: -1, F: builtinFold},
	{Name: "format", Arity: -1, F: builtinFormat},
	{Name: "isnil", Arity: 1, F: builtinIsnil},
	{Name: "len", Arity: 1, F: builtinLen},
	{Name: "load", Arity: 1, F: builtinLoad},
	{Name: "str", Arity: 1, F: builtinStr},
	{Name: "substr", Arity: 3, F: builtinSubstr},
	{Name: "typeof", Arity: 1, F: builtinTypeof},
}

// cond(b any, x any, y any) any
func builtinCond(args []Val, ctx *Ctx) (Val, error) {
	if args[0].Bool() {
		return args[1], nil
	}
	return args[2], nil
}

// contains(s string, substr string) bool
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

// error(s string) error
func builtinError(args []Val, ctx *Ctx) (Val, error) {
	switch s := args[0].(type) {
	case StringVal:
		return nil, fmt.Errorf(string(s))
	}
	return nil, fmt.Errorf("error: invalid argument type: %T", args[0])
}

// flatmap(f func('a)[]'b, xs []'a) []'b
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

// Three argument fold:
// fold(f func('a, 'b)'a, accu 'a, xs []'b ) 'a
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

// Two-argument fold:
// fold(f func('b, 'b)'b, xs []'b ) 'b
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

// format(fmt string, args ...any) string
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

// isnil(x any) bool
func builtinIsnil(args []Val, ctx *Ctx) (Val, error) {
	_, ok := args[0].(NilVal)
	return BoolVal(ok), nil
}

// len(x any) int
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

// builtinLoad loads a module (file) and stores it in the context.
// It returns the module body as a Val.
// load(name string) any
func builtinLoad(args []Val, ctx *Ctx) (Val, error) {
	name, ok := args[0].(StringVal)
	if !ok {
		return nil, fmt.Errorf("load: expected string argument got: %s", args[0])
	}
	lmod, err := LoadModule(string(name), ctx.dropLocals())
	if err != nil {
		return nil, err
	}
	return lmod.AsRec(), nil
}

// str(x any) string
func builtinStr(args []Val, ctx *Ctx) (Val, error) {
	return StringVal(args[0].String()), nil
}

// substr(s string, start int, end int) string
func builtinSubstr(args []Val, ctx *Ctx) (Val, error) {
	switch s := args[0].(type) {
	case StringVal:
		start, ok := args[1].(IntVal)
		if !ok {
			return nil, fmt.Errorf("substr: 2nd argument must be an int, got %T", args[1])
		}
		end, ok := args[2].(IntVal)
		if !ok {
			return nil, fmt.Errorf("substr: 3nd argument must be an int, got %T", args[2])
		}
		if start < 0 || start > end || int64(end) > int64(len(s)) {
			return nil, fmt.Errorf("substr: invalid start(%d)/end(%d) arguments for string of length %d",
				start, end, len(s))
		}
		return StringVal(string(s)[start:end]), nil
	}
	return nil, fmt.Errorf("substr: invalid type: %T", args[0])
}

// typeof(x any) string
func builtinTypeof(args []Val, ctx *Ctx) (Val, error) {
	return StringVal(args[0].Typ().Id), nil
}
