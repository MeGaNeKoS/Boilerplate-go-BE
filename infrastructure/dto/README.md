# Data Transfer Objects (`dto`)

This directory defines simple structs used to move data between layers. Using DTOs keeps database or HTTP specific types from leaking across package boundaries. Common types such as `JWTUser` live at the package root, while subdirectories hold service specific models.

`JWTUser` holds claims extracted from access tokens. The `response` subpackage defines generic response structures and standardised error codes used by the REST layer.

Folder pattern:

- `item` – request/response models for the item service.
- `response` – generic HTTP response helpers shared across handlers.

Add new service folders as needed to keep unrelated DTOs separate.

Run the DTO tests with:

```bash
go test ./infrastructure/dto/...
```
