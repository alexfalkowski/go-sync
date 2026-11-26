# Accepted design for execution

Part of the [accepted-design index](../accepted-design.md).

## Hook errors and timeouts

`(*Hook).Error` calls `OnError` when the supplied error is non-nil and the
handler is configured. It returns the handler's result in that case and the
supplied error otherwise. Execution helpers pass it the error from `OnRun`.

`Wait` passes the caller's context to `OnRun`. Its own timeout only bounds the
wait. When the timeout or caller cancellation wins, `Wait` returns `nil` and
`OnRun` may continue. `Timeout` passes a derived context to `OnRun` and returns
its cancellation cause when that context ends first. `OnRun` must observe that
context to stop in response to cancellation. Preserve these contracts when
changing either helper.

[Hook and timeout implementation](../../../sync.go) ·
[Timeout and cancellation tests](../../../sync_test.go)

## Worker scheduling and completion

`Schedule` and `TrySchedule` report only scheduling errors. Handler errors go
through `Hook.Error` for `OnError` side effects. The scheduling context also
goes to the handler. For a separate execution budget that starts when the
handler runs, derive a context inside `OnRun`.

`Worker.Wait` uses a goroutine per call to observe completion. A canceled wait
can return before that goroutine observes completion, even if the handlers
have already finished. The observer remains until the scheduled handlers
finish. The handlers' own contexts govern their cancellation.

[Worker implementation](../../../worker.go) ·
[Scheduling and completion tests](../../../worker_test.go)

## Future and Task cancellation

`Async` and `Go` start their callbacks even when the work context is already
done. Callbacks are responsible for observing that context. The `Await` context
only bounds the caller's wait. `Done` signals completion after the result is
published, and later callers can retrieve the cached result through `Await`.

[Future and Task implementation](../../../future.go) ·
[Caching and cancellation tests](../../../future_test.go)

## ErrorsGroup batches

`ErrorsGroup` joins non-nil errors in submission order and retains them for
the group's lifetime. Use a fresh group for each independent batch. A zero
concurrency limit prevents new work from starting, matching `errgroup`.

[ErrorsGroup implementation](../../../group.go) ·
[Aggregation and limit tests](../../../group_test.go)
