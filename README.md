# fugu-ocean

## Running locally

Bring up Postgres, MinIO, and Meilisearch:

```sh
docker compose up -d postgres minio minio-init meilisearch
```

Apply database migrations (uses `config.yaml`'s Postgres settings by default;
override with `CONFIG=path/to/config.yaml` or the usual `POSTGRES_*` env vars):

```sh
make migrate-up      # apply all pending migrations
make migrate-status   # show applied/pending migrations
make migrate-down     # roll back the most recent migration
make migrate-create NAME=add_users_table   # scaffold a new migration
```

Run the API:

```sh
go run ./cmd/api config.yaml
```

Check it's wired up to its dependencies:

```sh
curl http://localhost:8080/health
```

`/health` returns `200` with `{"status":"ok", ...}` when Postgres, MinIO, and
Meilisearch are all reachable, or `503` with the failing check(s) called out
otherwise.

## Development checks

Run the test suite with:

```sh
make test
```

Linting uses the pinned `golangci-lint` version. Install it locally with:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.0
```

Then run:

```sh
make lint
```

`make lint` uses the binary in `$(go env GOPATH)/bin` by default. Set `GOLANGCI_LINT` to use a different path.
