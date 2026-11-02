# apperr

Coded errors for Go applications. Lower layers return errors with a stable
code, a message, an optional cause and extra fields; handlers read the code
back with `CodeOf` and answer accordingly. Sentinels `ErrInvalidInput`,
`ErrConflict` and `ErrForbidden` mark the usual kinds of rejection.

## Install

```sh
go get github.com/ez-technologies/apperr
```

## Usage

```go
const CodeItemTaken apperr.Code = "item_taken"

func init() { apperr.Register(CodeItemTaken) }

func reserve(id int) error {
	return apperr.Wrap(CodeItemTaken, apperr.ErrConflict).With("item_id", id)
}

// In the handler:
err := reserve(7)
if errors.Is(err, apperr.ErrConflict) {
	// answer 409 with apperr.CodeOf(err) and apperr.ExtraOf(err)
}
```

`apperr.Codes()` lists the registered codes, sorted, for example to check in a
test that a client translates every one.

## Licence

MIT
