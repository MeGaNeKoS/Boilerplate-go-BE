# Validation Helpers (`validator`)

This package wraps [go-playground/validator](https://github.com/go-playground/validator).
In addition to the library's built in rules it registers `notBeforeNow` and
`notAfterNow` date checks. Call `ValidateStruct` to validate a struct and
receive a `ValidationErrors` value when validation fails. The helper translates
`validator.ValidationErrors` into human friendly messages so HTTP handlers can
return clear error responses.

Run the validator tests with:

```bash
go test ./pkg/validator/...
```
