# Utility Helpers (`utils`)

This directory contains small helper functions that are not tied to any single
layer.  Packages here are safe to import from the service layer, infrastructure
or server layers without introducing extra dependencies.  Most helpers are thin
wrappers that keep common logic in one place so it can be shared by the
transports and the use cases.

## `context`

Wrappers for storing and retrieving common values on `context.Context`.  These
helpers keep the context keys private to this package while providing simple
`Set*` and `Get*` functions for:

- the current database transaction
- the authenticated user information
- a request scoped logger
- repository and outbound aggregators

Handlers in the `server` layer set these values so the `services` can
retrieve them later without importing infrastructure packages.

## `generic`

Miscellaneous helpers including `ReplacePlaceholders` for template-style string
substitution.  The unexported `boolToInt` function is used internally by the
replacement logic and in tests.

## `path`

`NormalizeBasePath` ensures configured base paths start and end with a slash. It is used when building REST routes and documentation.

## `http`

Functions for creating standardised HTTP success and error responses based on
`pkg/code` values and the configured application name.  The `GenerateErrorResponse`
and `GenerateSuccessResponse` helpers ensure all endpoints return a consistent
payload structure.

## `jwt`

A small JWT service that can sign and validate tokens using the configured RSA
key pair. Initialise it with `InitializeJWTService` then retrieve the singleton
with `GetJWTService`.  Tokens created by this helper embed the current
environment name so they cannot be used across environments.

## `secure`

Defines `SecureString` and `SecureBytes` aliases that mask values when
formatted. If the value is shorter than 16 characters the entire string is
replaced with asterisks. Longer values show the first and last four characters
with the length in the middle so secrets are not leaked.

## `utils.go`

Contains `UniqueIdByTime` which generates random hexadecimal identifiers.
Tests override the `randReader` variable so failures can be simulated.

Run the helper tests with:

```bash
go test ./utils/...
```
