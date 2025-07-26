# REST Server (`server/rest`)

This folder contains the HTTP transport built with `chi`. Routes are grouped in the
`routes` package, handlers under `handlers` act as the controllers, and
`middleware` sets up logging, authentication and context values for each request.

The generated OpenAPI documentation is served under `/docs`. Route definitions
are stored separately in `routes/*_def.go` so examples and descriptions only
need to be maintained in one place. Keeping them out of the handler files
avoids the limitations of annotation-based docs, letting us include structured
examples without duplicating code.
