# Accepted design for aliases and wrappers

Part of the [accepted-design index](../accepted-design.md).

## Type aliases

Keep the convenience aliases as true Go type aliases so callers retain type
identity with the standard library and `golang.org/x/sync`. This applies to
`Once`, `Mutex`, `RWMutex`, `WaitGroup`, `ErrorGroup`, the atomic aliases
including `Pointer[T]`, `Semaphore`, and the `Any*` aliases.

[Alias overview](../../../doc.go) ·
[Semaphore implementation](../../../semaphore.go)

## Typed storage and singleflight

`Map[K,V]`, `Pool[T]`, `Value[T]`, and `SingleFlightGroup[T]` use type assertions
at their underlying storage or result boundary. Preserve stored types and
nil-interface behavior. `Map` exposes nil interface keys and values as the
type's zero value while preserving presence where the API reports it. `Value`
inherits `atomic.Value`'s rejection of nil interface stores and its concrete
type constraints. `SingleFlightGroup` exposes a nil interface result as the
zero value of `T` and deduplicates only in-flight calls.

[Map](../../../map.go) · [Pool](../../../pool.go) · [Value](../../../atomic.go) ·
[SingleFlightGroup](../../../group.go)

## Buffer pool capacity

`BufferPool.Put` resets the buffer's length and keeps its capacity. Callers own
the capacity policy and can discard oversized buffers. Treat re-pooling a
buffer at its current capacity as expected behavior in reliability reviews.

[BufferPool implementation](../../../bytes.go)
