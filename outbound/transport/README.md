# Transport Helpers (`outbound/transport`)

Generic helpers used by outbound clients. `HTTPOutbound` builds and executes HTTP
requests, while `GRPCOutbound` wraps a simple gRPC invocation. Both types take a
`logger.Logger` to trace requests and return structured responses.
