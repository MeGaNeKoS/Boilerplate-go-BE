# Repositories (`repositories`)

Repositories contain the data access logic for the services. The `Repository` aggregator lazily constructs individual repositories using a shared database and caches them for reuse.

Each specific repository lives in its own subfolder and exposes an interface that can be used by the service layer without importing infrastructure code.
