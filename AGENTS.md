# AGENTS.md

This repo is `github.com/alexfalkowski/go-sync`, a small Go library whose package name is `sync`. It provides focused concurrency helpers: hook-based wait/timeout helpers, bounded worker scheduling, errgroup/singleflight helpers, typed pool/map/atomic wrappers, and convenience aliases for sync and atomic primitives.

## Shared Standards

Use the shared `coding-standards` skill from `bin/skills/coding-standards` for code changes, bug fixes, refactors, reviews, tests, linting, documentation, PR summaries, commits, Makefile changes, CI validation, and verification.

## Setup Notes

- The `bin` git submodule is required. The root `Makefile` only includes `bin/build/make/go.mak` and `bin/build/make/git.mak`, so most `make` targets fail without it.
- Initialize the submodule with `git submodule sync && git submodule update --init`; after `bin` exists, `make submodule` is also available.
- `go.mod` declares `go 1.26.0`. The code uses newer APIs such as `sync.WaitGroup.Go` and test APIs such as `t.Context()`.
- CI uses CircleCI with `alexfalkowski/go:3.17`.

## Layout

All library code is in the repo root as package `sync`.

- `doc.go` has the package overview and exported API semantics.
- `sync.go`, `worker.go`, and `group.go` contain the hook, timeout, worker, errgroup, and singleflight helpers.
- `pool.go`, `bytes.go`, `atomic.go`, and `map.go` contain typed wrappers and aliases.
- Tests are black-box tests in package `sync_test`.
- `test/reports/` is for generated reports and artifacts.

## Commands

CI runs:

```sh
make clean
make dep
make lint
make sec
make specs
make benchmark
make coverage
make codecov-upload
```

Use the narrowest relevant target for local validation:

- `make dep` runs `go mod download`, `go mod tidy`, and `go mod vendor`.
- `make specs` runs tests through `gotestsum` with `-race`, vendored deps, JUnit, and coverage profile output.
- If `gotestsum` is unavailable, `go test ./...` is the simplest fallback, but it is not equivalent to CI specs.
- `make lint` runs field alignment and `golangci-lint` when installed.
- `make fix-lint` applies configured golangci-lint fixes.
- `make sec` runs `govulncheck -show verbose -test ./...`.
- `make coverage` writes filtered coverage outputs under `test/reports/`.

## API Conventions

- Keep exported behavior stable; this is a small public library.
- Public APIs should remain documented in GoDoc and, when helpful, in README examples.
- Keep tests in external package `sync_test` and use `github.com/stretchr/testify/require`.
- `Hook` error handling is centralized in `(*Hook).Error`.
- `Wait` is best-effort: timeout or context cancellation returns `nil` and does not stop `OnRun`.
- `Timeout` uses a derived context and returns `context.Cause` on timeout/cancellation.
- `Worker.Schedule` returns only scheduling errors; handler errors are routed through `OnError`.
- Alias types (`Once`, `Mutex`, `RWMutex`, `WaitGroup`, `ErrorGroup`, atomics, `Pointer[T]`) must remain true aliases.
- Generic wrappers (`SingleFlightGroup[T]`, `Map[K,V]`, `Pool[T]`, `Value[T]`) use type assertions; avoid changing stored types or nil-interface semantics casually.

## Gotchas

- Run `make dep` before validation or code changes unless the task is purely read-only.
- Do not hand-edit generated reports, vendored code, or lockfile/vendor state unless dependency work requires it.
- `make specs` and `go test ./...` are different; prefer the repo target when validating behavior.
- Only rely on optional tools (`gotestsum`, `golangci-lint`, `govulncheck`, `codecovcli`, etc.) when the target you run actually invokes them.
