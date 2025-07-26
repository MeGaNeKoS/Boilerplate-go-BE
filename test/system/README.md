# System Service Tests (`test/system`)

Integration tests for the system diagnostic endpoints. A server instance is started with stubbed dependencies and requests are sent with `httptest`. These tests also run when neither `grpc` nor `kafka` tags are present.
