# Async Package Conventions

`gopkg/common/async` provides shared concurrency primitives for Go modules across this monorepo.

This package owns generic mechanics only:
- `Future[T]` and `Promise[T]`
- async constructors and combinators such as `Async`, `AsyncWithContext`, `Map`, `Then`, `All`, `Any`, and `Race`
- task pool helpers such as `Pool`, `Parallel`, `ParallelFirstError`, and `FirstSuccess`
- sentinel errors for cancellation and pool lifecycle outcomes

This package does not own:
- business workflow policy
- retry rules, scheduling, or queue semantics
- transport lifecycle behavior
- repository transaction or persistence policy
- logging, metrics, or alerting requirements

Keep exported APIs generic, preserve caller-controlled contexts, and avoid adding helpers that only make sense for one feature or one adapter.
