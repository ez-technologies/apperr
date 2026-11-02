// Command client calls the example server and words its refusals in two
// languages.
//
// The server answers a refusal with a code, its own English text and extra
// fields. The client keeps its own sentence for each code, in each language
// it supports, and falls back to the server's text for a code it does not
// know yet. This is why codes, not sentences, cross the wire: the wording
// belongs to the client, and changing it needs no server release.
//
//	go run ./examples/httpserver &
//	go run ./examples/client
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// translations is what a client ships: a sentence per code per language.
// Placeholders in braces are filled from the error's extra fields.
var translations = map[string]map[string]string{
	"en": {
		"name_required":  "Please give the item a name.",
		"name_too_long":  "The name can be at most {max_length} characters.",
		"item_not_found": "Item {item_id} does not exist.",
		"item_taken":     "Someone else has already reserved item {item_id}.",
		"not_owner":      "You can only release an item you reserved.",
	},
	"fr": {
		"name_required":  "Veuillez donner un nom à l'article.",
		"name_too_long":  "Le nom peut contenir au plus {max_length} caractères.",
		"item_not_found": "L'article {item_id} n'existe pas.",
		"item_taken":     "Quelqu'un d'autre a déjà réservé l'article {item_id}.",
		// "not_owner" is missing on purpose: the check below finds it, and
		// the reader sees the server's text instead.
	},
}

type errorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Extra   map[string]any `json:"extra"`
}

// translate words a refusal in lang, or returns the server's text for a code
// the client does not know.
func translate(lang string, e errorBody) string {
	text, ok := translations[lang][e.Code]
	if !ok {
		return e.Message
	}
	for k, v := range e.Extra {
		text = strings.ReplaceAll(text, "{"+k+"}", fmt.Sprint(v))
	}
	return text
}

// missing lists, per language, the server's codes the client cannot word.
func missing(codes []string) map[string][]string {
	out := map[string][]string{}
	for lang, words := range translations {
		for _, c := range codes {
			if _, ok := words[c]; !ok {
				out[lang] = append(out[lang], c)
			}
		}
	}
	return out
}

func main() {
	base := flag.String("server", "http://localhost:8080", "address of the example server")
	flag.Parse()

	var codes []string
	if err := get(*base+"/codes", &codes); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("server codes: %s\n", strings.Join(codes, ", "))
	for _, lang := range []string{"en", "fr"} {
		if m := missing(codes)[lang]; len(m) > 0 {
			fmt.Printf("missing in %s: %s\n", lang, strings.Join(m, ", "))
		}
	}
	fmt.Println()

	steps := []struct {
		title, path, user, body string
	}{
		{"add an item with no name", "/items", "", `{"name":""}`},
		{"add an item with a long name", "/items", "", `{"name":"a very long name for one item"}`},
		{"add an item", "/items", "", `{"name":"lamp"}`},
		{"reserve it as alice", "/items/1/reserve", "alice", ""},
		{"reserve it as bob", "/items/1/reserve", "bob", ""},
		{"release it as bob", "/items/1/release", "bob", ""},
		{"reserve a missing item", "/items/9/reserve", "alice", ""},
	}
	for _, s := range steps {
		status, e, err := post(*base+s.path, s.user, s.body)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> %d\n", s.title, status)
		if e.Code == "" {
			continue
		}
		fmt.Printf("  code %s\n", e.Code)
		fmt.Printf("  en   %s\n", translate("en", e))
		fmt.Printf("  fr   %s\n", translate("fr", e))
	}
}

func get(url string, into any) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(into)
}

func post(url, user, body string) (int, errorBody, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		return 0, errorBody{}, err
	}
	req.Header.Set("X-User", user)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, errorBody{}, err
	}
	defer resp.Body.Close()
	var e errorBody
	if resp.StatusCode >= 400 {
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			return 0, errorBody{}, err
		}
	}
	return resp.StatusCode, e, nil
}
