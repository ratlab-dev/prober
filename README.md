# Prober[Not Recommended for production. Do your research before using]

Prober is a health and performance monitoring tool designed to periodically probe and report the status of various infrastructure components, including S3-compatible object stores, MySQL databases, Kafka clusters, and Redis (standalone and cluster) instances. It is intended for use in environments where continuous verification of service availability and latency is critical.

## Features
- Periodic read/write probes for S3, MySQL, Kafka, and Redis (standalone and cluster)
- Configurable probe intervals and targets via YAML configuration
- Success/failure metrics for each probe
- Extensible architecture for adding new probe types

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
Edit the `config.yaml` file to define the clusters and probe settings for S3, MySQL, and Redis. Example configuration sections are provided in the file.
We plan Kafka support, it is work in progress

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


Or, if you want to use Docker Compose to spin up dependencies:

```powershell
docker-compose up
```

## Project Structure
- `cmd/` - Main entry point for the prober application
- `internal/probe/` - Probe logic for each supported service
- `config.yaml` - Example configuration file
- `docker-compose.yml` - Example Docker Compose setup for dependencies

## Extending
To add a new probe type, implement the `Prober` interface in a new package under `internal/probe/` and register it in `probe.go`.


## Bugs
- The redis cluster included in docker-compose does not work

## License
This project is licensed under the MIT License.
