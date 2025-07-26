# REST Handlers (`server/rest/handlers`)

Handlers convert HTTP requests into service calls and return standardised JSON
responses. Each handler expects context values such as repositories, outbound
clients and a logger to be provided by middleware.
