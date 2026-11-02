package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func call(t *testing.T, h http.Handler, path, user, body string) (int, errorBody) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("X-User", user)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var e errorBody
	if rec.Code >= 400 {
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&e))
	}
	return rec.Code, e
}

func TestHandler_AnswersRefusalsWithStatusCodeAndExtra(t *testing.T) {
	h := newHandler(newStore())

	status, e := call(t, h, "/items", "", `{"name":""}`)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, CodeNameRequired, e.Code)

	status, e = call(t, h, "/items", "", `{"name":"a very long name for one item"}`)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, CodeNameTooLong, e.Code)
	assert.Equal(t, map[string]any{"max_length": float64(maxNameLength)}, e.Extra)

	status, _ = call(t, h, "/items", "", `{"name":"lamp"}`)
	assert.Equal(t, http.StatusCreated, status)

	status, _ = call(t, h, "/items/1/reserve", "alice", "")
	assert.Equal(t, http.StatusOK, status)

	status, e = call(t, h, "/items/1/reserve", "bob", "")
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, CodeItemTaken, e.Code)
	assert.Equal(t, "item is reserved by someone else: conflict", e.Message)

	status, e = call(t, h, "/items/1/release", "bob", "")
	assert.Equal(t, http.StatusForbidden, status)
	assert.Equal(t, CodeNotOwner, e.Code)

	status, e = call(t, h, "/items/9/reserve", "alice", "")
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, CodeItemNotFound, e.Code)
}

func TestHandler_HidesUncodedErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, assert.AnError)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"message":"internal error"}`, rec.Body.String())
}
