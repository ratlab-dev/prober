# Prober [Not Recommended for production. Do your research before using]

Prober is a health and performance monitoring tool designed to periodically probe and report the status of various infrastructure components, including S3-compatible object stores, MySQL databases, Kafka clusters, HTTP(S) endpoints, and Redis (standalone and cluster) instances. It is intended for use in environments where continuous verification of service availability and latency is critical.

## Features
- Periodic read/write probes for S3, MySQL, HTTP(S), and Redis (standalone and cluster)
- **Live config reload**: Prober watches `config.yaml` for changes and reloads only the affected probes, without restarting the service or unaffected probes
- **Config error resilience**: If the config is invalid, prober continues running with the last good config and logs persistent errors until fixed
- **Per-probe metadata**: All probes provide a human-readable `MetadataString()` for logging and debugging
- **Proxy support for HTTP probes**: Test HTTP(S) endpoints via a configurable proxy
- **Success/failure metrics** for each probe
- **Extensible architecture** for adding new probe types

## Use Case
Prober is ideal for DevOps, SRE, and platform teams who need to:
- Continuously monitor the health of critical infrastructure services
- Detect outages or performance degradation early
- Integrate probe results into dashboards or alerting systems

## Setup

### Prerequisites
- Go 1.18 or newer
- Docker (optional, for running dependencies or using docker-compose)

### Configuration
Edit the `config.yaml` file to define the clusters and probe settings for S3, MySQL, Redis, HTTP, and more. Example configuration sections are provided in the file.

- **HTTP probe**: Supports method, body, headers, proxy, and unacceptable status codes.
- **Live reload**: Any change to `config.yaml` is picked up automatically. Only the changed clusters are restarted.
- **Config errors**: If the config is invalid, prober logs the error every 30 seconds and continues with the last good config.

### Building
To build the prober binary:

```powershell
# From the project root
go build -o prober ./cmd
```

### Running
To run the prober:

```powershell
# From the project root
./prober config.yaml
```
The default config file would be located at ./config.yaml

If you want to use Docker Compose to spin up dependencies:

```powershell
docker-compose up
```

## Project Structure
- `cmd/` - Main entry point for the prober application
- `internal/probe/` - Probe logic for each supported service
- `config.yaml` - Example configuration file
- `docker-compose.yml` - Example Docker Compose setup for dependencies

## Extending
To add a new probe type, implement the `Prober` interface (including `MetadataString()`) in a new package under `internal/probe/` and register it in `probe.go`.

## Notes
- **Kafka probe is not yet working**: The Kafka probe is a placeholder and does not perform real health checks yet.
- **Redis probes are fixed**: Redis (standalone and cluster) probes are robust and support per-cluster live reload.
- **HTTP probe**: Fully supports proxy, custom headers, and status code validation.
- **Config reload**: Prober is resilient to config errors and will not stop running if the config is broken.
