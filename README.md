# go-pkg

Shared Go libraries for Embrace ID services. The repository is a **multi-module monorepo**: each top-level area is an independent Go module published under the `pkg.embrace.id/...` import namespace, so consumers depend only on the pieces they use.

All modules target **Go 1.26.1**.

## Modules

| Import path | Directory | Summary |
|---|---|---|
| `pkg.embrace.id/common` | `common/` | General utilities: async primitives, logging, pagination, pointer/time helpers, validation. |
| `pkg.embrace.id/platform/apperr` | `platform/apperr/` | Typed application errors keyed by an integer domain code. Zero non-stdlib dependencies. |
| `pkg.embrace.id/platform/cache` | `platform/cache/` | Cache interface with Redis and in-memory implementations (`.../cache/redis`, `.../cache/memory`). |
| `pkg.embrace.id/platform/connection-postgres` | `platform/connection-postgres/` | Postgres connection factory (GORM over the pgx driver). Package name is `postgresconn`. |

## Installation

Add the module you need to your project:

```bash
go get pkg.embrace.id/common
go get pkg.embrace.id/platform/apperr
go get pkg.embrace.id/platform/cache
go get pkg.embrace.id/platform/connection-postgres
```

Because the import host (`pkg.embrace.id`) is private, configure Go to skip the public proxy/checksum database for it:

```bash
go env -w GOPRIVATE=pkg.embrace.id
```

## Usage

### `common/async` — futures, promises, and worker pools

```go
import "pkg.embrace.id/common/async"

// Run work asynchronously and await it with context/timeout support.
f := async.Async(func() (User, error) { return fetchUser(id) })
user, err := f.Get(ctx)

// Bounded-concurrency task pool; Run collects every task error.
pool := async.NewPool(async.WithWorkers(4))
_ = pool.AddMany(taskA, taskB, taskC) // each is func() error
errs := pool.Run(ctx)
```

Also available: `Map`/`Then` for chaining, `All`/`Any`/`Race` for combining futures, `Promise[T]` for external completion, and `Parallel`/`FirstSuccess` helpers.

### `common/validation` — accumulating field validation

```go
import "pkg.embrace.id/common/validation"

bag := validation.NewBag()
bag.RequiredString("email", in.Email, "email is required")
bag.MaxStringLength("name", in.Name, 50, "name must be at most 50 characters")
bag.ULID("id", in.ID, "id must be a valid ULID")

if bag.HasAny() {
    return bag.Failures() // []validation.Failure with Code, Field, Message, Params
}
```

### `common/pagination`

```go
import "pkg.embrace.id/common/pagination"

p := pagination.NewPagination(page, limit) // clamps to DefaultLimit / MaximumLimit
rows := query.Offset(p.GetOffset()).Limit(p.Limit).Find(...)
p.SetPagination(total)
hasMore := p.HasNextPage()
```

### `common/pointer` and `common/timeparse`

```go
import (
    "pkg.embrace.id/common/pointer"
    "pkg.embrace.id/common/timeparse"
)

ptr := pointer.Val("hello")        // *string
val := pointer.Extract(ptr)        // "" if ptr is nil
opt := pointer.EmptyNil(0)         // nil, since the value is the zero value

// SliceVal preserves the nil-vs-empty distinction (nil slice -> nil pointer).
maybe := pointer.SliceVal([]int{}) // non-nil *[]int wrapping an empty slice

// Strict date parsing rejects non-canonical inputs (e.g. "2024-1-5").
t, ok := timeparse.ParseStrict("2024-01-15", timeparse.LayoutDate)
```

### `common/logger`

A package-level logrus singleton; use the package functions directly.

```go
import "pkg.embrace.id/common/logger"

logger.WithError(err).Error("failed to create user")
logger.WithFields(logrus.Fields{"user_id": id}).Info("user created")
```

### `platform/apperr` — typed domain errors

```go
import "pkg.embrace.id/platform/apperr"

const CodeUserNotFound = 1001

err := apperr.Wrap(CodeUserNotFound, sql.ErrNoRows)

if apperr.HasCode(err, CodeUserNotFound) { /* map to a 404, etc. */ }
code, ok := apperr.CodeOf(err) // extract the domain code from anywhere in the chain
```

`errors.Is` matches `*apperr.Error` values by code, and the wrapped cause stays inspectable via `errors.Unwrap`.

### `platform/cache` — Redis or in-memory

Both implementations satisfy the `cache.Cache` interface, so callers depend on the interface and swap the backend.

```go
import (
    "pkg.embrace.id/platform/cache"
    cacheredis "pkg.embrace.id/platform/cache/redis"
    cachemem "pkg.embrace.id/platform/cache/memory"
    "github.com/redis/go-redis/v9"
)

var c cache.Cache

// Redis-backed
c = cacheredis.NewRedisCache(redis.NewClient(&redis.Options{Addr: "localhost:6379"}))

// In-memory (useful for tests)
c = cachemem.NewMemoryCache()

_ = c.SetJSON(ctx, "user:1", user, time.Hour)
err := c.GetJSON(ctx, "user:1", &user) // returns cache.ErrNotFound when absent
```

### `platform/connection-postgres` — GORM client

```go
import postgresconn "pkg.embrace.id/platform/connection-postgres"

db, err := postgresconn.NewClient(postgresconn.Config{
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "postgres",
    Database: "app",
    SSLMode:  "disable",
    // MaxIdleConns / MaxOpenConns / ConnMaxLifetime / PingTimeout default
    // sensibly when left zero.
})
// db is a *gorm.DB; pass postgresconn.WithGormConfig(...) to customize.
```

## Development

There is no `go.work` file — run `go` commands from inside the relevant module directory.

```bash
# Work within one module
cd common && go test ./...
cd common && go test -run TestName ./...   # a single test
cd common && go vet ./... && go build ./...

# Test and vet every module from the repo root
for d in common platform/apperr platform/cache platform/connection-postgres; do
  (cd "$d" && go test ./... && go vet ./...)
done
```

`gofmt` and `go vet` are the baseline checks (there is no linter config or CI yet). After changing a module's imports, run `go mod tidy` in that module.

### Testing notes

Tests use [testify](https://github.com/stretchr/testify) and are table-driven. Redis tests run against an in-process [miniredis](https://github.com/alicebob/miniredis) — no external Redis is required. The Postgres success-path test connects to a local Postgres (configurable via the standard `PG*` environment variables) and **skips** when none is reachable.

## Versioning

Every module lives in a subdirectory (there is no module at the repo root), so Go pins each one through a **path-prefixed tag** matching the module directory:

- `common/vX.Y.Z`
- `platform/apperr/vX.Y.Z`
- `platform/cache/vX.Y.Z`
- `platform/connection-postgres/vX.Y.Z`

A plain `vX.Y.Z` tag marks the repository as a whole but does not, by itself, pin any subdirectory module. Until path-prefixed tags are published, `go get pkg.embrace.id/<module>` resolves to a pseudo-version of the latest commit.

## Conventions

See [CLAUDE.md](./CLAUDE.md) for the cross-cutting patterns these libraries follow (functional options, context validation, sentinel-error translation, error wrapping, defensive copying).
