package apperr

import (
	"errors"
	"sort"
	"sync"
)

// Code is the stable, machine-readable name of a refusal.
//
// A server answers an error with its text and, next to it, this code. A client
// can word the code in the reader's language and fall back to the text for a
// code it does not know. Applications declare their own codes as constants and
// register them with Register.
type Code string

var (
	mu         sync.RWMutex
	registered = map[Code]struct{}{}
)

// Register adds codes to the list returned by Codes. Registering a code twice
// has no effect.
func Register(codes ...Code) {
	mu.Lock()
	defer mu.Unlock()
	for _, c := range codes {
		registered[c] = struct{}{}
	}
}

// Codes lists every registered code, sorted. It can be compared against a
// client's list of translated codes in a test.
func Codes() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(registered))
	for c := range registered {
		out = append(out, string(c))
	}
	sort.Strings(out)
	return out
}

// Error is a refusal with a code. It reads as its message, unwraps to its
// cause so errors.Is still finds a sentinel behind it, and may carry extra
// fields for the answer.
type Error struct {
	Code    Code
	Message string
	Cause   error
	Extra   map[string]any
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Cause }

// New makes a refusal with a code and its text.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap gives an existing error a code, keeping its text and its chain.
func Wrap(code Code, cause error) *Error {
	return &Error{Code: code, Message: cause.Error(), Cause: cause}
}

// With adds a field the answer carries next to the code.
func (e *Error) With(key string, value any) *Error {
	if e.Extra == nil {
		e.Extra = map[string]any{}
	}
	e.Extra[key] = value
	return e
}

// CodeOf is the code an error carries, or "" for one without.
func CodeOf(err error) Code {
	var coded *Error
	if errors.As(err, &coded) {
		return coded.Code
	}
	return ""
}

// ExtraOf is what an error carries next to its code, or nil.
func ExtraOf(err error) map[string]any {
	var coded *Error
	if errors.As(err, &coded) {
		return coded.Extra
	}
	return nil
}
