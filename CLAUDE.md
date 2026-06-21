# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A collection of shared Go libraries for Embrace ID services (imported by downstream apps such as `cakapp`). There is no application or `main` package here — every package is a library meant to be imported under the `pkg.embrace.id/...` namespace.

## Module layout — this is a multi-module repo

There is **no `go.work` file**. Each top-level area is an independent Go module with its own `go.mod`, dependency set, and version. You must run `go` commands from inside the relevant module directory; a command run at the repo root will not see any module.

| Directory | Module path | Notes |
|---|---|---|
| `common/` | `pkg.embrace.id/common` | Pure utilities: `async`, `logger`, `pagination`, `pointer`, `timeparse`, `validation`. Depends only on logrus + testify. |
| `platform/apperr/` | `pkg.embrace.id/platform/apperr` | Typed domain errors. **Zero non-stdlib dependencies** — keep it that way. |
| `platform/cache/` | `pkg.embrace.id/platform/cache` | Cache interfaces (package `cache`) plus `redis/` and `memory/` implementation subpackages within the same module. |
| `platform/connection-postgres/` | `pkg.embrace.id/platform/connection-postgres` | GORM + pgx Postgres client. Directory name differs from the package name `postgresconn`. |

All modules target **Go 1.26.1**.

## Common commands

Run per-module (replace `common` with the target module dir):

```bash
cd common && go test ./...                     # test one module
cd common && go test -run TestName ./...        # single test
cd common && go vet ./... && go build ./...
cd common && gofmt -l .                         # list unformatted files
```

Test and vet every module from the repo root:

```bash
for d in common platform/apperr platform/cache platform/connection-postgres; do
  (cd "$d" && go test ./... && go vet ./...)
done

# After changing imports/deps in a module, tidy that module:
cd platform/cache && go mod tidy
```

There is currently no Makefile, CI workflow, or linter config (`.golangci.yml`); `gofmt`/`go vet` are the baseline checks. Adding a dependency requires editing that one module's `go.mod` only.

## Cross-cutting conventions

These patterns recur across packages — follow them when extending or adding code.

- **Functional options for constructors.** `NewClient(cfg, ...Option)`, `NewPool(...PoolOption)`, `NewRedisCache(client, ...RedisCacheOption)`. New constructors should take a config/required arg plus variadic options rather than long parameter lists.
- **Validate `context.Context` at the top of every method.** Cache implementations call a `validateContext`/`validateContextAndState` helper that rejects a nil context and a cancelled context (and a closed cache) before doing work. Mirror this in new context-taking methods.
- **Sentinel errors live in the contract package; implementations translate to them.** e.g. `cache.ErrNotFound`, `cache.ErrNilValue` are defined in package `cache`; the redis adapter maps `goredis.Nil → sharedcache.ErrNotFound`. Callers match on the shared sentinel, never the driver error.
- **Wrap errors with an operation prefix:** `fmt.Errorf("cache get: %w", err)`. Always use `%w` so the chain stays inspectable.
- **`apperr` carries an `int` domain code, not strings.** Use `apperr.New(code)` / `apperr.Wrap(code, cause)`, and match with `apperr.HasCode(err, code)` / `apperr.CodeOf(err)`. Its custom `Is` compares by code, so `errors.Is` works across the chain.
- **Defensive copying.** Byte slices stored in / returned from the memory cache and `Param` slices in `validation` are copied so callers can't mutate internal state. Preserve this when touching those paths.
- **Generics in `common`.** `async.Future[T]`/`Promise[T]`, `pointer.Val[T]`/`Extract[T]`. Note `pointer.SliceVal` and `pointer.EmptyNil` exist specifically to preserve the nil-vs-empty distinction — don't collapse them.
- **`validation.Bag` is nil-safe** (methods no-op / return zero on a nil receiver). `logger` is a package-level logrus singleton accessed via package functions (`logger.Info`, `logger.WithFields`).

## Testing conventions

Tests use `testify` (`require` for fatal preconditions, `assert` for value checks), are table-driven, and call `t.Parallel()` where safe. Redis tests run against an in-process `miniredis` (`miniredis.RunT(t)`) — no external Redis is needed.

## Commits

History follows Conventional Commits (e.g. `chore: import shared Go modules from cakapp_source`).
