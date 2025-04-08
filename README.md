# go-outbox
A sidecar service that runs alongside the main application in a separate container or process, forming a sidecar pattern deployment. It has access to the same database as the application and operates continuously to forward outbox events.


## Folders

`/cmd/outbox-sidecar/`    -> main package, parses flags/config and starts the service

`/pkg/config/`            -> configuration loading and validation

`/pkg/storage/`           -> outbox repository interfaces and implementations (postgres.go, mysql.go, mongo.go, etc.)

`/pkg/broker/`            -> broker interface and implementations (rabbitmq.go, pubsub.go, ...)

`/pkg/processor/`         -> the outbox processor logic (event loop, retry, DLQ handling)

`/pkg/telemetry/`         -> setup for OpenTelemetry (tracer, meter)

`/docs/`                  -> documentation and usage guides
