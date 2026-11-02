// Package apperr holds errors that cross layer boundaries.
//
// Validation often runs deep in the stack, inside repositories and
// transactions, far from the HTTP request. The sentinels here let a rejection
// travel back up to the handler with enough information to answer correctly,
// without the lower layers importing anything about HTTP.
package apperr

import "errors"

// ErrInvalidInput marks a rejection caused by what the client sent.
//
// The request was understood and the server is healthy; the values in it were
// not acceptable. A handler typically answers 400 for this and 500 for
// anything else.
var ErrInvalidInput = errors.New("invalid input")

// ErrConflict marks a rejection because the data changed underneath the caller.
//
// The request was valid when it was written but is stale now. A handler
// typically answers 409, telling the client to reload rather than fix its input.
var ErrConflict = errors.New("conflict")

// ErrForbidden marks a rejection because the caller is not allowed to do this.
//
// A handler typically answers 403. A resource the caller may not even see is
// better answered as not found, so its existence is not confirmed.
var ErrForbidden = errors.New("forbidden")
