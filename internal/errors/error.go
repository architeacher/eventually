package errors

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
)

type (
	// Kind is the error classification.
	Kind uint8
	// Operation that was performed to produce this error.
	Operation string

	// Error is the implementation of the error interface.
	Error struct {
		// Op is the performed operation, usually it is the method name.
		// e.g. Bus.Publish()
		Op Operation
		// Kind of the error such as consumer not found
		// or "Other" if its class is unknown or irrelevant.
		Kind Kind
		// Err is the underlying error that has triggered this one, if any.
		Err error
	}
)

const (
	// Other is a fallback classification of the error.
	Other Kind = iota
	// Invalid entity is not allowed.
	Invalid
	// Exist item already exists.
	Exist
	// NotFound item is not found.
	NotFound
	// MinLength should be 1 or more.
	MinLength
	// Failure to apply an operation.
	Failure
	// Panic recovery errors.
	Panic
)

var (
	// Separator is used to separate nested errors.
	Separator = ":\n\t"
)

func (k Kind) String() string {
	switch k {
	case Invalid:
		return "invalid entity is provided, entity can not be nil"
	case Exist:
		return "Item already exists: %s"
	case NotFound:
		return "item not found"
	case MinLength:
		return "at least one entity, should be passed"
	case Failure:
		return "could not perform operation on data %s"
	case Panic:
		return "panic"
	}

	return "unknown error kind"
}

// E creates an error
func E(args ...interface{}) error {
	if len(args) == 0 {
		panic("call to errors.E with no arguments")
	}

	e := &Error{}
	for _, arg := range args {
		switch arg := arg.(type) {
		case Operation:
			e.Op = arg
		case Kind:
			e.Kind = arg
		case *Error:
			// Make a clone
			clone := *arg
			e.Err = &clone
		case error:
			e.Err = arg
		default:
			_, file, line, _ := runtime.Caller(1)
			log.Printf("errors.E: bad call from %s:%d: %v", file, line, args)

			return Errorf("unknown type %T, value %v in error call", arg, arg)
		}
	}

	return e
}

func (e *Error) Error() string {
	b := new(bytes.Buffer)

	if e.Op != "" {
		pad(b, ": ")
		b.WriteString(string(e.Op))
	}

	if e.Kind != 0 {
		pad(b, ": ")
		b.WriteString(e.Kind.String())
	}

	if e.Err != nil {
		// Indent on new line if we are cascading non-empty errors.
		if prevErr, ok := e.Err.(*Error); ok {
			if !prevErr.isZero() {
				pad(b, Separator)
				b.WriteString(e.Err.Error())
			}
		} else {
			pad(b, ": ")
			b.WriteString(e.Err.Error())
		}
	}

	if b.Len() == 0 {
		return "no error"
	}

	return b.String()
}

func (e *Error) isZero() bool {
	return e.Op == "" && e.Kind == 0 && e.Err == nil
}

// Errorf creates an error from a given format string.
func Errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// Is checks if the error of a given kind.
func Is(kind Kind, err error) bool {
	e, ok := err.(*Error)
	if !ok {
		return false
	}

	if e.Kind != Other {
		return e.Kind == kind
	}

	if e.Err != nil {
		return Is(kind, e.Err)
	}

	return false
}

// pad appends str to the buffer if the buffer already has some data.
func pad(b *bytes.Buffer, str string) {
	if b.Len() == 0 {
		return
	}

	b.WriteString(str)
}
