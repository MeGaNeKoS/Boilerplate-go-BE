# Outbound Integrations (`outbound`)

Outbound clients encapsulate calls to external systems so the services layer
remains transport agnostic. The `Outbound` aggregator lazily constructs these
clients and caches them for reuse.

Each remote service lives under `service/<name>` and exposes its own `Service`
interface. Helpers for common transports such as HTTP and gRPC are provided
under `transport`.
