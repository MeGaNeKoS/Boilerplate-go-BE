# REST Handlers (`server/rest/handlers`)

Handlers convert HTTP requests into service calls and return standardised JSON
responses. Each handler expects context values such as repositories, outbound
clients and a logger to be provided by middleware.

## Debugging error inference
Error inference logs how long reflection-based analysis takes for each handler
and package, which can help benchmark startup cost when experimenting with new
controllers.
