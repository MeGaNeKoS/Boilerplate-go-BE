# Test Suite (`test`)

Contains unit and integration tests for the project. Each service has a subfolder holding transport-specific tests. The `files` package exercises file upload/download handlers while `testutil` offers helpers for mocking dependencies. Run all tests with:

```bash
go test ./...
```
