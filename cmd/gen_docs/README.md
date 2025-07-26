# Generate Documentation (`gen_docs`)

This command builds the OpenAPI specification without starting the server. It reads the configuration file if available and writes `openapi.json` and `openapi.yaml` into the `docs` directory.

Usage:

```bash
go run ./cmd/gen_docs [config path]
```

If no path is provided `config.yaml` is used by default.
