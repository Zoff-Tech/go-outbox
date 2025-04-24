<style>
p {
margin-left:40px;
width:80%;
text-align: justify;
}
</style>


# Golang Sidecar Service for the Outbox Pattern
## Overview

The [Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html#:~:text=A%20service%20command%20typically%20needs,13%20and%20publish%20an%20event) is a microservice design pattern that ensures atomicity between database operations and message publishing​.
​

* The outbox addresses the problem of reliably updating data and sending events without using distributed transactions. In this pattern, the application writes domain changes and an event record to an outbox table in the same transaction. 
* A separate process then reads from the outbox and publishes events to a message broker, guaranteeing that if the database commit succeeds, the event will eventually be delivered​

​

This **sidecar service** is a **Go** implementation of the `Outbox Pattern`, designed to run alongside a primary application. 
It monitors the outbox table for new events and relays them to a messaging system with robust guarantees. 

### Key Features:

**Pluggable storage layer:** Supports multiple databases (PostgreSQL, MySQL, MongoDB, etc.) behind a common interface.

**Extensible messaging brokers:** Supports RabbitMQ and Google Cloud Pub/Sub initially, with a flexible interface for others.

**Event-driven processing:** Uses event notifications from the database when possible, falling back to polling if needed.

**Configuration management:** Offers YAML and environment variable config with structured parsing and validation.

**Observability:** Integrates with OpenTelemetry for distributed tracing and metrics following industry best practices.

**Reliability:** Implements retries with exponential backoff, ensures idempotent message handling, and uses dead-letter queues for failures.

**Kubernetes-ready:** Runs as a sidecar container in the same Pod as the application, sharing lifecycle and resources.

## Architecture and Design


Outbox Pattern architecture: 

![](./docs/image.png)

The service’s business data and an outbox event are saved atomically **(1)**. 
<br/>
A separate sidecar process then reads the outbox **(2)** and publishes the event to a message broker **(3)**, ensuring reliable, eventual delivery. 
<br/>
<p>
The sidecar service runs alongside the main application in a separate container or process, forming a sidecar pattern deployment.
It has access to the same database as the application and operates continuously to forward outbox events. The main application, after performing its business logic (for example, creating an order), writes a new event record to the outbox table as part of the same transaction that modifies domain data. Once committed, the sidecar service will detect the outbox entry and publish a corresponding message to the configured message broker. This decoupling guarantees atomicity and prevents the dual-write problem where a database update could succeed but the message send fails (or vice versa)​
(<a href="https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html) 
[dev.to](https://dev.to/trojan295/">docs.aws.amazon.com</a>)  (<a href="https://dev.to/trojan295/transactional-outbox-pattern-example-in-golang-and-mongodb-36mb#:~:text=One%20problem%20I%20had%20was,for%20example%2C%20an%20application%20crash">dev.to</a>)
​</p>
<br/>


In this design, the `Outbox Sidecar` is **event-driven** by default: it subscribes to notifications of new outbox records or change events from the database. If the underlying storage engine doesn’t support push notifications or if notifications are lost, the service falls back to polling the outbox table at a configurable interval. This ensures no events are missed even in absence of events or on crashes. The sidecar shares the same lifecycle as the main app – *“for each instance of the application, an instance of the sidecar is deployed and hosted alongside it”​* ([learn.microsoft.com](https://learn.microsoft.com/en-us/azure/architecture/patterns/sidecar#:~:text=A%20sidecar%20service%20isn%27t%20necessarily,deployed%20and%20hosted%20alongside%20it))


This co-location minimizes communication latency and allows the sidecar to reliably use local environment/configuration identical to the app’s. 

### Major components of the service include:
* **Outbox Repository Interface:** Abstracts the storage operations (querying new events, updating statuses, etc.) so multiple databases can be supported.
* **Message Broker Interface:** Abstracts the messaging system, enabling integration with different brokers by implementing a common publish API.
* **Outbox Processor:** Core engine that ties everything together, using either DB events or polling to retrieve outbox messages and dispatch them to a broker.
* **Configuration Loader:** Reads YAML or environment variables into a config struct, validating required fields.
* **Observability Hooks:** Instrumentation points for tracing and metrics (e.g. span around message processing, counters for processed messages)
* **Retry & DLQ Logic:** Handles retrying message sends with backoff, marking messages as sent or directing them to dead-letter channels after failures.

**For more Implementation datails read my blog post: here**

## Project Folders

`/cmd/outbox-sidecar/`    -> main package, parses flags/config and starts the service

`/pkg/config/`            -> configuration loading and validation

`/pkg/storage/`           -> outbox repository interfaces and implementations (postgres.go, mysql.go, mongo.go, etc.)

`/pkg/broker/`            -> broker interface and implementations (rabbitmq.go, pubsub.go, ...)

`/pkg/processor/`         -> the outbox processor logic (event loop, retry, DLQ handling)

`/pkg/telemetry/`         -> setup for OpenTelemetry (tracer, meter)

`/docs/`                  -> documentation and usage guides


