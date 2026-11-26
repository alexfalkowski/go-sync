# Accepted design

This index records go-sync's accepted design choices and tradeoffs. The area
files carry the same weight as [AGENTS.md](../../AGENTS.md).

Read the relevant files before proposing a change or recording a finding or
audit entry. Revisit a decision when the owner asks or new evidence changes its
documented contract or tradeoff.

- [Execution](accepted-design/execution.md) covers hooks, deadlines, worker
  scheduling and completion, asynchronous results, and error aggregation.
- [Aliases and wrappers](accepted-design/wrappers.md) covers type aliases,
  singleflight, typed storage, and buffer capacity.
