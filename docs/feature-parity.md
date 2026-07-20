# jaflow Feature Parity

## Purpose

This document defines the feature-parity target for the current port from the
Python/shell `tw-flow` implementation in `jacazul-ai-cli` to Go.

Parity means preserving observable workflow contracts, safety behavior, state
transitions, isolation boundaries, and actionable errors. It does not mean
copying the Python structure or its internal implementation.

Reference tests:

```text
~/source/jacazul-ai/jacazul-ai-cli/master/tests
```

## Current Phase Rule

This is a temporary rule for the porting phase only:

```text
Red -> Green -> Refactor -> Mutation Check -> Parity Review
```

1. Write a valid failing contract test.
2. Implement the smallest behavior that makes it pass.
3. Refactor command boundaries and reduce accidental complexity.
4. Break the protected rule intentionally and verify the test fails.
5. Compare observable behavior with the reference test and implementation.

A green build without meaningful parity coverage is not feature completion.

## Reference Test Audit

The reference suite contains candidate tests for the following core/runtime
areas:

| Area | Reference tests | Parity treatment |
|---|---|---|
| Project isolation | `core_test.py`, `test_project_identity.py` | Port with temporary project state and cross-project leak checks |
| Lifecycle | `flow_test.py` | Port plan, dependencies, execute, outcome, done, reopen, discard, and handoff |
| Focus and sessions | `flow_test.py`, `test_session_list.py` | Port global focus, task stack, smart focus, independent sessions, heartbeat, and fallback |
| Context | `flow_test.py` | Port structured annotations, inherited context, notes, and ticket inheritance |
| Cache | `cache_test.py` | Port storage, TTL, cached signals, force refresh, selective invalidation, and session isolation |
| Dashboard | `flow_test.py`, `test_recently_closed.py` | Port status, ponder, plans, filters, history, and output contracts |
| Backlog | `test_backlog.py` | Port hide/show/activate behavior and state markers |
| Roadmap | `test_roadmap_init_guard.py` | Port duplicate-ledger protection for pending and completed phases |
| Safety | `security_test.py`, `flow_test.py` | Port workflow vaccination, outcome gates, and actionable errors |
| Ticket boundary | selected `flow_test.py` tests | Port metadata persistence and inherited ticket awareness; provider brokers remain external |

The initial inventory identified 104 candidate core/runtime tests:

- core: 3
- flow: 41
- cache: 29
- security: 6
- backlog: 5
- recently closed: 4
- roadmap guard: 3
- session list: 9
- project identity: 4

## Test Quality Findings

These reference tests must not be copied blindly:

- `test_backlog.py` and `test_recently_closed.py` do not inject isolated
  `TASKDATA`/`JACAZUL_HOME`; rewrite them with temporary state before porting.
- `flow_test.py::test_vaccinated_done_enforces_python_quality` writes and stages
  a source file; replace it with a sandboxed fake quality-gate contract.
- `test_recently_closed_no_extra_requests` cannot prove internal request count
  from one subprocess invocation; replace it with an observable request seam.
- Several tests ignore setup command exit codes and rely only on substring
  assertions; strengthen them with state assertions and mutation checks.
- Tests that only assert a directory exists, a substring appears, or a command
  returns zero need a deliberate mutation to prove their oracle is meaningful.

A test is valid for parity only when it has an explicit contract, isolated
fixtures, a meaningful oracle, and fails under a relevant mutation.

## Implementation Backlog

Implement in this order, keeping one command boundary per feature slice:

1. Isolated Go contract-test harness for project, session, cache, filesystem,
   and external-command state.
2. Command registry, global options, process-boundary errors, and project-scoped
   state foundation.
3. Plan/task lifecycle: creation, dependency readiness, execute, outcome,
   done, reopen, discard, and handoff.
4. Focus and context: global focus, task stack, smart focus, independent
   sessions, annotations, inherited context, and ticket metadata.
5. Dashboard and cache: status, ponder, plans, cache TTL, cached signals,
   force refresh, selective invalidation, and session isolation.
6. Backlog, roadmap, recently closed history, and duplicate-ledger guards.
7. Safety and parity hardening: actionable errors, workflow restrictions,
   mutation checks, and regression coverage.
8. Final review of complexity, security, documentation, and Go 1.25/1.26
   validation.

## Command Architecture

Follow the `nvimim` CLI pattern:

```text
cmd/jaflow/main.go       parser, global options, command registry, process exit
internal/cli/             one command type and Execute method per command
internal/workflow/        small shared workflow behavior
internal/storage/         project/session persistence boundaries
internal/cache/           cache policy and invalidation
```

Do not translate the Python `FlowManager` into a Go monolith. Feature parity is
measured per command and per contract, not by matching the old class layout.

## Out of Scope for jaflow Core

Do not port these as jaflow workflow features:

- JIT prompt generation and the `hatch` subsystem.
- Persona rendering and client-specific agent artifacts.
- Pi, Claude, Copilot, Gemini, or OpenCode launchers and bootstraps.
- GitHub/Bitbucket broker implementation internals.

The workflow engine may expose stable boundaries for clients, tickets, and
brokers, but those integrations remain owned by their respective systems.

## Completion Criteria

A parity slice is complete only when:

- its reference contract has a Go test;
- the test starts red against the missing behavior;
- the implementation passes the test;
- relevant mutations make the test fail;
- fixtures cannot touch real project or user state;
- command complexity remains local and reviewable;
- `gofmt`, `go test`, `go test -race`, and `go vet` pass;
- behavior and residual differences are documented.
