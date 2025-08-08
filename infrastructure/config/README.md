# Configuration

The `config` package loads YAML settings from a file into the `config.Cfg` variable. It is used by the entry points under `cmd` to configure the service. `LoadConfig()` reads the file once during startup, populating the package-level variable so other packages can access it. YAML was chosen for readability and to keep configuration separate from code.

A default `config.yaml` is provided at the repository root and a fully annotated example lives in `examples/config.yaml`.

The `OpenAPI` section controls documentation settings such as the spec version
(`3.0.3` or `3.1.0`) and the paths where the generated docs are served. For
example:

```yaml
OpenAPI:
  Version: "3.1.0"
  Docs:
    Public: "/docs/public"
    Internal: "/docs/internal"
```

Load the configuration by passing the file path via the `--config` flag:

```bash
go run . --config ./examples/config.yaml
```

Run the package tests with:

```bash
go test ./config/...
```
