# Go Code Review Guide

Use this guide when reviewing changes in Fugu Ocean. Prioritize correctness,
clarity, security, and maintainability over personal style preferences.

The primary references are:

- Effective Go
- Go Code Review Comments
- Go Proverbs
- Go standard-library conventions

## Review priorities

Classify findings by impact:

- P0 — security issue, data loss/corruption, panic, broken build, or unavailable core flow.
- P1 — incorrect behavior, broken contract, missing authorization, resource leak, or likely production failure.
- P2 — maintainability or reliability issue that should be fixed before merging.
- P3 — optional improvement; do not block a change on it.

Do not report style-only feedback when `gofmt`, `go vet`, or golangci-lint can enforce it.

## Go idioms

Prefer:

- Small, focused functions with clear names.
- Simple control flow and early returns over deep nesting.
- Concrete types until an interface is required by a consumer.
- Interfaces defined close to their consumer, not pre-emptively.
- The zero value being useful where practical.
- Standard-library solutions before adding a dependency.
- Context passed as the first argument to operations that can block or perform I/O.
- Errors returned to the caller instead of panics for expected failures.

Avoid:

- Unnecessary abstractions, factories, global state, and reflection.
- `interface{}` / `any` where a concrete type is known.
- Boolean parameters that make a function call unclear.
- Exported identifiers without a useful reason.
- Mutable package-level state.
- Goroutines without a defined owner, cancellation path, and lifecycle.

## Error handling

Check that:

- Every meaningful error is handled or deliberately documented as ignored.
- Errors include useful context and preserve the cause with `%w` when callers may need `errors.Is` or `errors.As`.
- Sentinel errors are used for stable, meaningful categories; callers do not match error strings.
- External input and infrastructure failures do not cause panics.
- HTTP/API errors do not leak internal details or secrets.
- Cleanup failures are handled when they can affect correctness.

## Context, I/O, and concurrency

Check that:

- Database, storage, search, and HTTP operations receive `context.Context`.
- Context is not stored in a struct or replaced with `context.Background()` in request flows.
- Resources such as response bodies, rows, files, transactions, and multipart uploads are closed or rolled back.
- Timeouts and cancellation are respected around external calls.
- Shared mutable state is synchronized and free from races.
- Background workers have bounded retries, clear ownership, and observable failures.

## API and domain behavior

Check that:

- Input is validated at the HTTP boundary and business rules live in the service layer.
- Authentication and authorization are enforced server-side for every protected action.
- Repository implementations do not leak storage-specific details into services.
- State transitions are explicit and valid.
- Database changes that must succeed together are transactional.
- New configuration fields are validated and documented.
- Sensitive values are not logged or returned in API responses.

## Tests

A change should include tests for observable behavior, not implementation details.

Review for:

- A failing test before the relevant bug fix or feature implementation when practical.
- Success, validation failure, and authorization failure cases.
- Boundary values and malformed external input.
- Tests that invoke public behavior rather than private helpers where possible.
- Regression tests that would fail if the important protection were removed.
- Deterministic tests: no real network calls, wall-clock dependency, or shared environment leakage.

## Review comments

A good comment explains the consequence and suggests a direction:

> P1: This path accepts an empty JWT secret, so the service can start with
> predictable authentication credentials. Reject an empty or too-short secret
> during configuration loading.

Do not block a change for naming or formatting preferences unless they obscure
meaning, conflict with established project conventions, or affect correctness.

If there are no blocking findings, say so explicitly and list any P3 suggestions
separately.
