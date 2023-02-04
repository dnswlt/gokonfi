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
}

func builtinLen(args []Val) (Val, error) {
	switch arg := args[0].(type) {
	case StringVal:
		return IntVal(len(arg)), nil
	case *RecVal:
		return IntVal(len(arg.Fields)), nil
	}
	return nil, fmt.Errorf("invalid type for len: %T", args[0])
}

func builtinContains(args []Val) (Val, error) {
	switch s := args[0].(type) {
	case StringVal:
		if substr, ok := args[1].(StringVal); ok {
			return BoolVal(strings.Contains(string(s), string(substr))), nil
		}
		return nil, fmt.Errorf("invalid type for arg #2 of contains: %T", args[1])
	}
	return nil, fmt.Errorf("invalid argument types for contains: (%T, %T)", args[0], args[1])
}

func builtinCond(args []Val) (Val, error) {
	if args[0].Bool() {
		return args[1], nil
	}
	return args[2], nil
}

func builtinFormat(args []Val) (Val, error) {
	if len(args) == 0 {
		return StringVal(""), nil
	}
	if len(args) == 1 {
		return args[1], nil
	}
	format, ok := args[0].(StringVal)
	if !ok {
		return nil, fmt.Errorf("first argument must be a format string, got %T", args[0])
	}
	formatArgs := make([]any, len(args[1:]))
	for i, arg := range args[1:] {
		formatArgs[i] = arg
	}
	s := fmt.Sprintf(string(format), formatArgs...)
	return StringVal(s), nil
}

func builtinStr(args []Val) (Val, error) {
	return StringVal(args[0].String()), nil
}
