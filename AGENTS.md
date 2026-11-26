# AGENTS.md

This repo is `github.com/alexfalkowski/go-sync`, a small Go library whose package name is `sync`. It provides focused concurrency helpers: hook-based wait/timeout helpers, bounded worker scheduling, typed async result handles (Future), errgroup/singleflight helpers, typed pool/map/atomic wrappers, and convenience aliases for sync and atomic primitives.

## Shared guidance

Use `bin/AGENTS.md` for shared skills and cross-repository defaults.

## Accepted design

Read the [accepted-design index](docs/agents/accepted-design.md) and the relevant
area files before proposing a change or recording a finding or audit entry.
Apply the recorded scope of each decision and add owner-accepted decisions to
the relevant area file as they are agreed.

## Setup Notes

- The `bin` git submodule is required. The root `Makefile` only includes `bin/build/make/go.mak` and `bin/build/make/git.mak`, so most `make` targets fail without it.
- Initialize the submodule before using shared Make targets. Use
  `make submodule` once the shared `bin` checkout is present; see
  `bin/AGENTS.md` for fresh-clone bootstrap details.
- `go.mod` owns the current toolchain declaration. The code may use newer Go
  APIs, so check `go.mod` before changing compatibility assumptions.
- CI uses CircleCI; see `.circleci/config.yml` for the build environment.

## Layout

All library code is in the repo root as package `sync`.

- `doc.go` has the package overview and exported API semantics.
- `sync.go`, `worker.go`, and `group.go` contain the hook, timeout, worker, errgroup, and singleflight helpers.
- `future.go` contains `Future`/`Async` and the error-only `Task`/`Go` helpers.
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

- `make dep` refreshes dependencies and vendor state.
- `make specs` runs the repository test target with reports and coverage input.
- `make lint` runs repository linting.
- `make fix-lint` applies configured lint fixes.
- `make sec` runs repository security checks.
- `make coverage` writes filtered coverage outputs under `test/reports/`.

## API Conventions

- Keep exported behavior stable; this is a small public library.
- Public APIs should remain documented in GoDoc and, when helpful, in README examples.
- Keep tests in external package `sync_test` and use `github.com/stretchr/testify/require`.

## Gotchas

- Run `make dep` before validation or code changes unless the task is purely read-only.
- Do not hand-edit generated reports, vendored code, or lockfile/vendor state unless dependency work requires it.
- Prefer `make specs` when validating behavior.
- Only rely on optional tools (`gotestsum`, `golangci-lint`, `govulncheck`, `codecovcli`, etc.) when the target you run actually invokes them.
