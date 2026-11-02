package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranslate_FillsExtraAndFallsBackToServerText(t *testing.T) {
	e := errorBody{Code: "item_taken", Message: "item is reserved by someone else: conflict", Extra: map[string]any{"item_id": 1}}
	assert.Equal(t, "Someone else has already reserved item 1.", translate("en", e))

	unknown := errorBody{Code: "not_owner", Message: "server text"}
	assert.Equal(t, "server text", translate("fr", unknown))
}

func TestMissing_ListsCodesWithoutTranslation(t *testing.T) {
	m := missing([]string{"item_taken", "not_owner"})
	assert.Empty(t, m["en"])
	assert.Equal(t, []string{"not_owner"}, m["fr"])
}
