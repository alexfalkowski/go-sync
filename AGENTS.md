# AGENTS.md

This repository is **`github.com/alexfalkowski/go-sync`**, a small Go library (package name: **`sync`**) that provides focused concurrency helpers: hook-based wait/timeout helpers, bounded worker scheduling, errgroup/singleflight helpers, typed pool/map/atomic wrappers, and convenience aliases for sync and atomic primitives.

## Shared skill

Use the shared `coding-standards` skill from `./bin/skills/coding-standards`
for cross-repository coding, review, testing, documentation, and PR
conventions. Treat this `AGENTS.md` as the repo-specific companion to that
skill.

## Repo prerequisites / setup

### Git submodule is required

The root `Makefile` is only:

- `Makefile:1-2` → includes `bin/build/make/go.mak` and `bin/build/make/git.mak`

Those files live in the **`bin` git submodule** (`.gitmodules:1-3`). If the submodule is not initialized, most `make` targets will fail.

Initialize it with either:

```sh
git submodule sync && git submodule update --init
```

or (available once the include is present):

```sh
make submodule
```

### Go toolchain version

`go.mod:1-6` declares `go 1.25.0`.

The code uses APIs that are not available in older Go versions (for example `sync.WaitGroup.Go` in `worker.go`, and `t.Context()` in tests such as `sync_test.go`). Use a Go toolchain that supports those APIs.

CI runs in a container image `alexfalkowski/go:2.102` (`.circleci/config.yml:4-7`).

## Code layout

All library code is at the repo root (single Go package: `sync`):

- `doc.go` – package overview and high-level semantics for the exported API.
- `sync.go` – aliases `Once`, `Mutex`, `RWMutex`; `Hook` (OnRun/OnError) and top-level helpers `Wait`, `Timeout`, `IsTimeoutError`.
- `group.go` – aliases `WaitGroup`, `ErrorGroup`; generic `SingleFlightGroup`.
- `worker.go` – `Worker` with bounded scheduling and a `Wait()`.
- `pool.go`, `bytes.go` – generic pool + `bytes.Buffer` pool.
- `atomic.go` – typed atomic aliases plus the generic `Value[T]` wrapper around `atomic.Value`.
- `map.go` – generic wrapper around `sync.Map`.

Tests are mostly written as black-box tests in package `sync_test`:

- `*_test.go` at repo root (e.g., `sync_test.go`, `worker_test.go`, `map_test.go`).

The `test/` directory is primarily used for **test artifacts/reports** (`test/reports/*`) produced by Make targets.

## Essential commands (what CI runs)

CircleCI (`.circleci/config.yml:19-56`) runs these in order:

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

Notes:

- `make dep` is defined in `bin/build/make/go.mak:25-26` and runs `go mod download`, `go mod tidy`, and `go mod vendor`.
- `make specs` is defined in `bin/build/make/go.mak:61-64` and runs tests via `gotestsum` with `-race` and coverage enabled.

## Build / test / lint details

### Tests

- Primary test entrypoint used by CI: `make specs` (`bin/build/make/go.mak:61-64`).
  - Writes JUnit to `test/reports/specs.xml`.
  - Writes a coverage profile to `test/reports/profile.cov`.
  - Uses `-mod vendor` and builds a `-coverpkg=...` list computed from the repo’s Go source files.

If you do not have `gotestsum` installed, `make specs` will fail; in that case, running `go test ./...` is the simplest fallback (it won’t generate the same reports).

### Coverage

Coverage-related targets (from `bin/build/make/go.mak`):

- `make coverage` (`bin/build/make/go.mak:84-86`) generates:
  - `test/reports/final.cov` via `make remove-generated-coverage` (`bin/build/make/go.mak:73-75`), which uses `bin/quality/go/covfilter`.
  - `test/reports/coverage.html` via `make html-coverage` (`bin/build/make/go.mak:76-79`).
  - Function coverage output via `make func-coverage` (`bin/build/make/go.mak:80-83`).

`.gocov` controls what gets filtered out; in this repo it contains `test` (`.gocov:1`).

### Lint / formatting

- `make lint` (`bin/build/make/go.mak:51-53`) runs:
  - field alignment check (`make field-alignment` → `bin/build/go/fa`)
  - `golangci-lint` (`make golangci-lint` → `bin/build/go/lint run ...`)

`bin/build/go/lint` only invokes `golangci-lint` **if it is installed** (`bin/build/go/lint:5-7`).

`.golangci.yml` enables a broad linter set and enables formatters (`gofmt`, `gofumpt`, `goimports`, `gci`). Auto-fixing is wired via:

```sh
make fix-lint
```

(`bin/build/make/go.mak:54-55`), which runs `golangci-lint ... --fix`.

`make format` exists (`bin/build/make/go.mak:57-60`) and runs `go fmt ./...`.

### Security

- `make sec` (`bin/build/make/go.mak:95-98`) runs `govulncheck -show verbose -test ./...`.

## Conventions & patterns seen in the code

- Public API is small and is tested from `sync_test` (external test package), so keep exported behavior stable.
- `Hook` error handling is centralized in `(*Hook).Error` in `sync.go`. `Wait`, `Timeout`, and `Worker.Schedule` all route handler errors through it.
- `Wait` is intentionally “best effort”: on timeout or `ctx.Done()` it returns `nil`. `Timeout` returns `ctx.Err()` on timeout/cancel. `Worker.Schedule` only returns scheduling errors and routes handler errors to `OnError`.
- The alias types (`Once`, `Mutex`, `RWMutex`, `WaitGroup`, `ErrorGroup`, `Int32`, `Int64`, `Uint32`, `Uint64`, `Uintptr`, `Bool`, `Pointer[T]`) should remain true aliases; changing them to wrappers would be a breaking API change.
- Generics are used throughout (`SingleFlightGroup[T]`, `Map[K,V]`, `Pool[T]`, `Value[T]`), and several wrappers use type assertions; avoid changing stored types or you may introduce panics.

## Tooling dependencies (observed)

These are invoked by Make targets and/or scripts in the `bin` submodule:

- Go tools: `go`, `govulncheck`, `golangci-lint`, `fieldalignment`, `gotestsum`, `codecovcli`.
- Misc tools used by less-common Make targets in the shared makefile: `mkcert`, `goda`, `dot`, `gsa`, `scc` (`bin/build/make/go.mak:109-129`).

Only rely on a tool if you can see it being called from a target you’re using.

## CI configuration

- CircleCI: `.circleci/config.yml`
  - Uses `make source-key` (from `bin/build/make/git.mak:176-177`) to create a `.source-key` file for cache keys.
  - Stores test results and artifacts from `test/reports/`.

## Common gotchas

- **Submodule required**: without `bin/`, `make` will fail because the root `Makefile` only includes make fragments.
- **Go version matters**: this repo’s `go.mod` and usage of newer APIs require a matching Go toolchain.
- **Specs vs plain go test**: `make specs` runs with `-race`, vendored deps, and writes reports; `go test ./...` is not equivalent.
