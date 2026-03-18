# Configuration

The `config` package loads YAML settings from a file into the `config.Cfg` variable. It is used by the entry points under `cmd` to configure the service. `LoadConfig()` reads the file once during startup, populating the package-level variable so other packages can access it. YAML was chosen for readability and to keep configuration separate from code.

A default `config.yaml` is provided at the repository root and a fully annotated example lives in `examples/config.yaml`.

The `OpenAPI` section controls documentation settings such as the spec version
(`3.0.3` or `3.1.0`), the paths where the generated docs and schemas are
served, and which HTML renderer is used. The `Servers` subsection sets the
public and internal base URLs used in generated links so `$id` fields and
`describedby` headers are absolute. For example:

```yaml
OpenAPI:
  Version: "3.1.0"
  Docs:
    Public:
      url: "/docs/public"
      schema: "/schema/v1"
    Internal:
      url: "/docs/internal"
      schema: "/schema"
    Renderer: "stoplight"
  Servers:
    Public: "https://api.example.com"
    Internal: "https://internal.example.com"
```

The `Server` section can optionally provide `InternalCIDRs`, a list of network
blocks that should be treated as internal. Requests from these ranges will get
links to internal schemas. When running behind a reverse proxy, the service
also honors the `X-Forwarded-For` and `X-Real-IP` headers when determining the
client IP.

Load the configuration by passing the file path via the `--config` flag:

```bash
go run . --config ./examples/config.yaml
```

Run the package tests with:

```bash
go test ./config/...
```
