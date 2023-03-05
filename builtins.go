package gokonfi

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
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
	{Name: "lptime", Arity: 1, F: builtinLenientParseTime},
	{Name: "load", Arity: 1, F: builtinLoad},
	{Name: "mkrec", Arity: -1, F: builtinMkrec},
	{Name: "pcall", Arity: -1, F: builtinPcall},
	{Name: "regexp_extract", Arity: -1, F: builtinRegexpExtract},
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
	return nil, &ValError{V: args[0]}
}

func pcallResult(value Val, err bool) Val {
	return NewRecWithFields(map[string]Val{
		"value": value,
		"err":   BoolVal(err),
	})
}

// From Lua: call f with optional args. Pass through the return value
// if f does not raise an error. Otherwise, return the error.
// pcall(f func, [arg any]*) any
func builtinPcall(args []Val, ctx *Ctx) (Val, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("pcall: expect at least one (function) argument")
	}
	f, ok := args[0].(CallableVal)
	if !ok {
		return nil, fmt.Errorf("pcall: 1st argument must be a callable, got %s", args[0].Typ().Id)
	}
	v, err := f.Call(args[1:], ctx)
	if err != nil {
		var valErr *ValError
		// Try to unwrap ValueError.
		// Note that errors may be chained, our Eval routine does not pass ValueErrors through unchanged.
		if errors.As(err, &valErr) {
			return pcallResult(valErr.V, true), nil

		}
		return nil, err
	}
	return pcallResult(v, false), nil
}

// flatmap(f func('a)[]'b, xs []'a) []'b
func builtinFlatmap(args []Val, ctx *Ctx) (Val, error) {
	f, ok := args[0].(CallableVal)
	if !ok {
		return nil, fmt.Errorf("flatmap: 1st argument must be a callable, got %s", args[0].Typ().Id)
	}
	xs, ok := args[1].(ListVal)
	if !ok {
		return nil, fmt.Errorf("flatmap: 2nd argument must be a list, got %s", args[1].Typ().Id)
	}
	result := []Val{}
	for _, x := range xs.Elements {
		fx, err := f.Call([]Val{x}, ctx)
		if err != nil {
			return nil, fmt.Errorf("flatmap: call failed: %w", err)
		}
		if ys, ok := fx.(ListVal); ok {
			// f returned a list (as it should): append all elements to the result.
			result = append(result, ys.Elements...)
		} else {
			// Value returned by f was not a list: append single value.
			result = append(result, ys)
		}
	}
	return ListVal{Elements: result}, nil
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
		return nil, fmt.Errorf("fold: 1st argument must be a callable, got %T", args[0])
	}
	xs, ok := args[2].(ListVal)
	if !ok {
		return nil, fmt.Errorf("fold: 3nd argument must be a list, got %T", args[1])
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
	// We expect the right number of arguments here, since this function is not exposed.
	f, ok := args[0].(CallableVal)
	if !ok {
		return nil, fmt.Errorf("fold: 1st argument must be a callable, got %T", args[0])
	}
	xs, ok := args[1].(ListVal)
	if !ok {
		return nil, fmt.Errorf("fold: 2nd argument must be a list, got %T", args[1])
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
	case ListVal:
		return IntVal(len(arg.Elements)), nil
	}
	return nil, fmt.Errorf("len: invalid type: %T", args[0])
}

func builtinLenientParseTime(args []Val, _ *Ctx) (Val, error) {
	s, ok := args[0].(StringVal)
	if !ok {
		return nil, fmt.Errorf("load: expected string argument got: %s", args[0].Typ().Id)
	}
	layouts := []string{
		"2006-01-02 15:04:05 -0700",       // // YYYY-MM-DD HH:MM:SS with timezone offset
		"2006-01-02 15:04:05",             // YYYY-MM-DD HH:MM:SS
		"2006-01-02T15:04:05Z07:00",       // ISO 8601
		"2006-01-02",                      // YYYY-MM-DD
		"Mon, 02 Jan 2006 15:04:05 -0700", // RFC1123 with numeric zone
	}
	for _, l := range layouts {
		if tm, err := time.Parse(l, string(s)); err == nil {
			r := NewRec()
			r.setField("year", IntVal(tm.Year()), nil)
			r.setField("month", IntVal(tm.Month()), nil)
			r.setField("day", IntVal(tm.Day()), nil)
			r.setField("hour", IntVal(tm.Hour()), nil)
			r.setField("minute", IntVal(tm.Minute()), nil)
			r.setField("second", IntVal(tm.Second()), nil)
			r.setField("nanosecond", IntVal(tm.Nanosecond()), nil)
			_, offset := tm.Zone()
			r.setField("offset", IntVal(offset), nil)
			return r, nil
		}
	}
	return nil, fmt.Errorf("could not parse time %q", s)
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

// The constructor for records. Useful to generate dynamic records
// whose field names are only known at runtime.
// mkrec(f string, fv any [, f string, fv any]*) record
func builtinMkrec(args []Val, ctx *Ctx) (Val, error) {
	if len(args) == 1 {
		lv, ok := args[0].(ListVal)
		if !ok {
			return nil, fmt.Errorf("mkrec: 1-argument version expects a list argument, got %s", args[0].Typ().Id)
		}
		return recFromList(lv.Elements)
	}
	return recFromList(args)
}

func recFromList(xs []Val) (*RecVal, error) {
	// Expect list of pairs of field name and field value.
	if len(xs)%2 != 0 {
		return nil, fmt.Errorf("mkrec: expected an even number of elements [field name, field value]*")
	}
	r := NewRec()
	for i := 0; i < len(xs)/2; i++ {
		f, ok := xs[i*2].(StringVal)
		if !ok {
			return nil, fmt.Errorf("mkrec: expected string at list index %d, got %s", i*2, xs[i*2].Typ().Id)
		}
		r.setField(string(f), xs[i*2+1], nil)
	}
	return r, nil
}

// regexp_extract(s string, regexp string [, group_index int]) string
func builtinRegexpExtract(args []Val, ctx *Ctx) (Val, error) {
	if len(args) != 3 && len(args) != 2 {
		return nil, fmt.Errorf("regexp_extract: invalid number of arguments: %d", len(args))
	}
	sv, ok := args[0].(StringVal)
	if !ok {
		return nil, fmt.Errorf("regexp_extract: 1st argument must be a string, got %s", args[0].Typ().Id)
	}
	s := string(sv)
	regexpStr, ok := args[1].(StringVal)
	if !ok {
		return nil, fmt.Errorf("regexp_extract: 2nd argument must be a string, got %s", args[1].Typ().Id)
	}
	group_index := 0
	if len(args) == 3 {
		if gi, ok := args[2].(IntVal); !ok {
			return nil, fmt.Errorf("regexp_extract: 3rd argument must be an int, got %s", args[2].Typ().Id)
		} else if int(gi) < 0 {
			return nil, fmt.Errorf("regexp_extract: group_index must be >= 0, got %d", gi)
		} else {
			group_index = int(gi)
		}
	}
	re, err := regexp.Compile(string(regexpStr))
	if err != nil {
		return nil, fmt.Errorf("regexp_extract: %w", err)
	}
	if group_index == 0 {
		r := re.FindString(s)
		return StringVal(r), nil
	}
	i := group_index * 2
	xs := re.FindStringSubmatchIndex(s)
	if xs == nil || i >= len(xs) || xs[i] < 0 {
		return StringVal(""), nil // No match
	}
	r := s[xs[i]:xs[i+1]]
	return StringVal(r), nil
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
