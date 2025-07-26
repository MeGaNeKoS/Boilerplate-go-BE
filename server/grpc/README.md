# gRPC Server (`server/grpc`)

This transport exposes the Item service defined in `proto/item.proto`. The server uses interceptors for panic recovery and request logging, and converts request metadata into context values for authentication and logging.

Compile the project with the `grpc` build tag to include this server.
