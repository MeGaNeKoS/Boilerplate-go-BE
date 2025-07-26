# Service Packages

This layer contains all business logic. Each **first-level folder** in this directory represents a service we provide (also known as a bounded context). Code here is independent from infrastructure concerns and only depends on other service packages and the generic utilities under `pkg`.
Two example services are included:

- **`item`** – demonstrates CRUD style interactions. The package defines a service with business rules while relying on repository interfaces from `repositories` and the outbound interface from `outbound`. Models live under `infrastructure/dto/item`.
- **`system`** – exposes small utility operations used by the REST API for health checks and debugging, e.g., returning the process ID or intentionally crashing.

Service implementations return error values using the `pkg/code` types. Those codes are later translated into transport specific responses by the server layer. Repository implementations live under `repositories` and outbound clients under `outbound`.

Run the service tests with:

```bash
go test ./services/...
```

See the root `README.md` for an overview of how the service layer fits into the rest of the project.
