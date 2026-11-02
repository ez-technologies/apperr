package apperr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	codeNoSuchItem   Code = "no_such_item"
	codeItemTaken    Code = "item_taken"
	codeNameRequired Code = "name_required"
)

func TestCodes_ListsRegisteredCodesSortedOnce(t *testing.T) {
	Register(codeNoSuchItem, codeItemTaken)
	Register(codeNameRequired, codeItemTaken)

	assert.Equal(t, []string{"item_taken", "name_required", "no_such_item"}, Codes())
}

func TestError_KeepsCauseCodeAndExtra(t *testing.T) {
	cause := errors.New("that item is already taken")
	err := Wrap(codeItemTaken, cause).With("item_id", 7)

	assert.Equal(t, cause.Error(), err.Error())
	assert.True(t, errors.Is(err, cause))
	assert.Equal(t, codeItemTaken, CodeOf(err))
	assert.Equal(t, map[string]any{"item_id": 7}, ExtraOf(err))
	assert.Equal(t, Code(""), CodeOf(errors.New("plain")))
	assert.Nil(t, ExtraOf(errors.New("plain")))
}

func TestError_FoundThroughWrapping(t *testing.T) {
	err := fmt.Errorf("saving: %w", Wrap(codeNameRequired, ErrInvalidInput))

	assert.True(t, errors.Is(err, ErrInvalidInput))
	assert.Equal(t, codeNameRequired, CodeOf(err))
}

func TestNew_HasCodeAndMessage(t *testing.T) {
	err := New(codeNoSuchItem, "no such item")

	assert.Equal(t, "no such item", err.Error())
	assert.Equal(t, codeNoSuchItem, CodeOf(err))
	assert.Nil(t, errors.Unwrap(err))
	assert.Nil(t, ExtraOf(err))
}
