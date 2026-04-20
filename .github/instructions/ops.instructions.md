---
applyTo: "Makefile,Dockerfile,docker-compose.yaml,configs/**"
description: "Local development operations. Use when starting, stopping, debugging, or operating the service and its infrastructure locally."
---

# Local Operations

## Startup Sequence

Infrastructure must be running before the application starts. Always follow this order:

```bash
make docker-up                    # 1. Start infrastructure
docker compose ps                 # 2. Verify all services show "healthy"
make migrate                      # 3. Apply database migrations (if postgres enabled)
make run                          # 4. Build and start the application
```

If `docker compose ps` shows any service as "starting" or "unhealthy", wait and check
logs before proceeding:

```bash
docker compose logs <service>     # Check a specific service
docker compose logs -f --tail=50  # Follow all logs
```

## Makefile Targets

| Target | What It Does |
|--------|-------------|
| `make build` | Compile binary to `./bin/server` |
| `make test` | Run all tests with `-race` and `-count=1` |
| `make run` | Build + run with `APP_ENV=local` |
| `make vet` | Run `go vet ./...` |
| `make lint` | Run `golangci-lint` (must be installed separately) |
| `make tidy` | Run `go mod tidy` with correct proxy settings |
| `make docker-up` | Start all infrastructure services in background |
| `make docker-down` | Stop all infrastructure services |
| `make docker-logs` | Tail logs from all services |
| `make docker-build` | Build the application Docker image |
| `make migrate` | Run SQL migrations against local Postgres |

## Configuration

YAML config files live in `configs/`:

| Path | Purpose |
|------|---------|
| `configs/base/*.yaml` | Default config, always loaded |
| `configs/local/app.yaml` | Local overrides (loaded when `APP_ENV=local`) |

Sensitive values (passwords, API keys) must come from environment variables, never YAML.
The config system merges `base/` + `<env>/` at startup.

## Ports

| Service | Default Port |
|---------|-------------|
| HTTP server | 8080 |
| gRPC server | 9090 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| Kafka | 9092 |

If a port conflict occurs, stop the conflicting process or change the port in
`docker-compose.yaml` and the corresponding `configs/base/*.yaml`.

## Troubleshooting

| Symptom | Diagnosis |
|---------|-----------|
| App fails to start | Check `docker compose ps` — infra may not be ready |
| Connection refused on 5432/6379/9092 | Run `make docker-up` — infra not running |
| Migration fails | Check `docker compose logs postgres` for errors |
| Tests fail with connection errors | Tests should not require infra — mock or use interfaces |
| `go mod tidy` fails | Run `make tidy` — it sets the correct proxy environment |
