package gokonfi

import (
	"errors"
	"fmt"
	"strings"
)

// The most generic error type returned by Konfi functions.
// This type should be used if no more specific error type
// is appropriate, but
// For root errors without a cause it's also fine to use Go's
// builtin error types.
type KonfiError struct {
	msg   string
	cause error
}

func (e *KonfiError) Error() string {
	if e.cause == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %s", e.msg, e.cause)
}

func (e *KonfiError) Unwrap() error {
	return e.cause
}

func chainError(cause error, format string, a ...any) error {
	return &KonfiError{msg: fmt.Sprintf(format, a...), cause: cause}
}

// FormattedError turns a (possible chain of) gokonfi errors
// such as EvalError or ParseError into a simple Go error
// with a nicely formatted (potentially multi-line) error message.
// All other error types are returned unchanged.
//
// In particular, the error message has human-readable indicators
// for the position at which the error(s) occurred, whenever possible.
func (ctx *Ctx) FormattedError(err error) error {
	msgs := []string{}
	for err != nil {
		switch e := err.(type) {
		case *KonfiError:
			msgs = append(msgs, e.msg)
		case *EvalError:
			p, ok := ctx.fileset().Position(e.Pos())
			if !ok {
				panic(fmt.Sprintf("cannot translate position %d", e.Pos()))
			}
			msgs = append(msgs, fmt.Sprintf("%s: %s", p.String(), e.msg))
		case *ParseError:
			p, ok := ctx.fileset().Position(e.Pos())
			if !ok {
				panic(fmt.Sprintf("cannot translate position %d", e.Pos()))
			}
			msgs = append(msgs, fmt.Sprintf("%s: %s", p.String(), e.msg))
		case *ScanError:
			p, ok := ctx.fileset().Position(e.Pos())
			if !ok {
				panic(fmt.Sprintf("cannot translate position %d", e.Pos()))
			}
			msgs = append(msgs, fmt.Sprintf("%s: %s", p.String(), e.msg))
		default:
			msgs = append(msgs, err.Error())
			break // Don't unwrap external errors.
		}
		err = errors.Unwrap(err)
	}
	return fmt.Errorf(strings.Join(msgs, "\n"))
}
