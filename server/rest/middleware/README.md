# REST Middleware (`server/rest/middleware`)

Custom `chi` middleware used by the HTTP server. It handles request logging,
recovery from panics and populates the context with repositories, outbound
clients and the authenticated token (as a `SecureString`).
