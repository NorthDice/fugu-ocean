# fugu-ocean

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
