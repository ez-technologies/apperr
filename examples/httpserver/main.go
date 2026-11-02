// Command httpserver is a tiny item store that answers refusals with codes.
//
// The store layer knows nothing about HTTP. It returns apperr errors that carry
// a code, a message and sometimes extra fields; the handler turns them into a
// status and a JSON body.
//
//	go run ./examples/httpserver
//	curl -s -X POST localhost:8080/items -d '{"name":""}'
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/ez-technologies/apperr"
)

// The codes this application can answer with. They are part of the API: a
// client words them, so they never change once published.
const (
	CodeNameRequired apperr.Code = "name_required"
	CodeNameTooLong  apperr.Code = "name_too_long"
	CodeItemNotFound apperr.Code = "item_not_found"
	CodeItemTaken    apperr.Code = "item_taken"
	CodeNotOwner     apperr.Code = "not_owner"
)

const maxNameLength = 20

func init() {
	apperr.Register(CodeNameRequired, CodeNameTooLong, CodeItemNotFound, CodeItemTaken, CodeNotOwner)
}

type item struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Reserved string `json:"reserved_by,omitempty"`
}

// store is the lower layer. It returns coded errors and never sees a request.
type store struct {
	mu    sync.Mutex
	items map[int]*item
	next  int
}

func newStore() *store { return &store{items: map[int]*item{}, next: 1} }

func (s *store) add(name string) (*item, error) {
	if name == "" {
		return nil, apperr.Wrap(CodeNameRequired, fmt.Errorf("name is required: %w", apperr.ErrInvalidInput))
	}
	if len(name) > maxNameLength {
		return nil, apperr.Wrap(CodeNameTooLong, fmt.Errorf("name is longer than %d characters: %w", maxNameLength, apperr.ErrInvalidInput)).
			With("max_length", maxNameLength)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	it := &item{ID: s.next, Name: name}
	s.items[it.ID] = it
	s.next++
	return it, nil
}

func (s *store) reserve(id int, user string) (*item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	if !ok {
		return nil, apperr.New(CodeItemNotFound, "no item with that id").With("item_id", id)
	}
	if it.Reserved != "" && it.Reserved != user {
		return nil, apperr.Wrap(CodeItemTaken, fmt.Errorf("item is reserved by someone else: %w", apperr.ErrConflict)).
			With("item_id", id)
	}
	it.Reserved = user
	return it, nil
}

func (s *store) release(id int, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	if !ok {
		return apperr.New(CodeItemNotFound, "no item with that id").With("item_id", id)
	}
	if it.Reserved != user {
		return apperr.Wrap(CodeNotOwner, fmt.Errorf("only the user who reserved an item can release it: %w", apperr.ErrForbidden)).
			With("item_id", id)
	}
	it.Reserved = ""
	return nil
}

// errorBody is what crosses the wire: the code for the client to word, the
// server's own text as a fallback, and any extra fields.
type errorBody struct {
	Code    apperr.Code    `json:"code,omitempty"`
	Message string         `json:"message"`
	Extra   map[string]any `json:"extra,omitempty"`
}

// statusOf maps an error to an HTTP status. The sentinel an error wraps decides
// the usual cases, so most new codes need no change here; a code that needs its
// own status is matched by name.
func statusOf(err error) int {
	switch {
	case errors.Is(err, apperr.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, apperr.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, apperr.ErrForbidden):
		return http.StatusForbidden
	}
	switch apperr.CodeOf(err) {
	case CodeItemNotFound:
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func writeError(w http.ResponseWriter, err error) {
	status := statusOf(err)
	body := errorBody{Code: apperr.CodeOf(err), Message: err.Error(), Extra: apperr.ExtraOf(err)}
	// Answer with the coded error's own text, not the context the handler added.
	var coded *apperr.Error
	if errors.As(err, &coded) {
		body.Message = coded.Message
	}
	if status == http.StatusInternalServerError {
		log.Printf("internal error: %v", err)
		body = errorBody{Message: "internal error"} // do not leak details
	}
	writeJSON(w, status, body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newHandler(s *store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /items", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeError(w, fmt.Errorf("reading body: %w", apperr.ErrInvalidInput))
			return
		}
		it, err := s.add(in.Name)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, it)
	})
	mux.HandleFunc("POST /items/{id}/reserve", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		it, err := s.reserve(id, r.Header.Get("X-User"))
		if err != nil {
			writeError(w, fmt.Errorf("reserving item %d: %w", id, err))
			return
		}
		writeJSON(w, http.StatusOK, it)
	})
	mux.HandleFunc("POST /items/{id}/release", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		if err := s.release(id, r.Header.Get("X-User")); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	// Every code the server can answer with, so a client can check it words them all.
	mux.HandleFunc("GET /codes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, apperr.Codes())
	})
	return mux
}

func main() {
	addr := flag.String("addr", "localhost:8080", "address to listen on")
	flag.Parse()
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, newHandler(newStore())))
}
