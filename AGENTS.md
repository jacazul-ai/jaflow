# jaflow Agent Guide

## Mission

`jaflow` is the Go-native migration of the workflow engine currently shipped
by `jacazul-ai-cli`. Its purpose is to replace the Python/shell `tw-flow`
stack with a portable local engine that can later support centralized
coordination.

The primary consumer is:

- `~/source/jacazul-ai/jacazul-ai-cli/master`
- <https://github.com/jacazul-ai/jacazul-ai-cli>

The reference implementation is the `jacazul/` package in that repository.
Use it to recover behavior and compatibility requirements, but do not copy
implementation accidents into Go without a reason.

## Runtime Context

The Jacazul distribution reaches agents through several client launchers:

- Pi
- Claude
- Copilot
- Gemini
- OpenCode

Each client has its own launcher/bootstrap path, but the runtime contract is
shared:

1. The environment bootstrap resolves the canonical `PROJECT_ID`, `TASKDATA`,
   session ID, mode, language, and Taskwarrior setup.
2. The client bootstrap makes the relevant Jacazul skills, agents, or
   extensions available to that client.
3. The client launcher starts the agent with the Jacazul runtime protocol.
4. The agent uses the workflow engine to preserve focus, plans, task context,
   annotations, session handoffs, and cached status output.

`jaflow` is the shared workflow engine in this chain. It must remain
independent of any specific client, persona, launcher, prompt format, or
agent bootstrap implementation.

## Explicit Non-Goals

Prompt and agent artifact generation is outside this repository's
responsibility. The `jacazul-ai-cli` `hatch` subsystem owns JIT prompt forging,
persona anchoring, and generated client-specific agent files. Do not move hatch
logic into `jaflow` or make the workflow engine depend on it. `jaflow` should
consume stable workflow inputs and expose workflow behavior to any client.

## Current Phase: Feature Parity

This repository is currently implementing feature parity with the existing
`tw-flow` behavior. The tests under
`~/source/jacazul-ai/jacazul-ai-cli/master/tests` are the primary behavioral
reference for this phase, especially:

- `tests/core_test.py`: executable entry points and strict project isolation.
- `tests/flow_test.py`: lifecycle, focus, context inheritance, tickets,
  dashboards, modes, handoffs, and completion safety.
- `tests/cache_test.py`: cache storage, TTL signals, force refresh, selective
  invalidation, filter-specific keys, and session isolation.
- `tests/security_test.py`: workflow vaccination, archive auditing, quoting,
  and raw Taskwarrior restrictions.
- `tests/test_session_list.py`, `tests/test_backlog.py`,
  `tests/test_recently_closed.py`, and `tests/test_roadmap_init_guard.py`:
  focused regressions for session, backlog, history, and roadmap behavior.

Parity rules for this phase:

- Port the existing behavioral tests or equivalent Go contract tests before
  declaring a feature complete.
- Use isolated temporary state in tests; never use the developer's real
  `TASKDATA`, home directory, cache, or network.
- Treat `internal/testharness` as fixture infrastructure, not feature proof.
  A harness test may verify that fixtures are isolated, but parity tests must
  exercise the real command/API and assert both sides of the boundary: project
  A must not observe project B's state, and project B must not observe project
  A's state.
- Preserve observable command behavior, safety restrictions, state transitions,
  cache boundaries, and actionable errors unless a deliberate compatibility
  decision is recorded.
- Do not add new workflow semantics merely because the Go implementation can
  support them. First match the reference behavior, then evolve it in a
  separately identified phase.
- A passing build without parity coverage is not feature completion.

### Phase-Scoped Development Loop

Only for the current porting and feature-parity phase, use a strict
Red-Green-Refactor loop:

1. **Red:** write a valid failing contract test for the reference behavior.
2. **Green:** implement the smallest change that makes the test pass.
3. **Refactor:** split command responsibilities and reduce accidental
   complexity without changing the contract.
4. **Mutation check:** intentionally break the protected rule, when practical,
   and verify the test fails rather than passing vacuously.
5. **Parity review:** compare the resulting observable behavior with the
   `jacazul-ai-cli` reference test and implementation.

This loop is a temporary rule for the migration phase. It must not be treated
as a permanent project-wide process requirement after parity is established.

## Migration Scope

The first compatibility target is the behavior exposed by:

- `tw-flow`: plan/task lifecycle, focus, status, ponder, context, notes,
  outcomes, handoffs, backlog, roadmap, sessions, and commit drafts.
- `taskp`: project-aware Taskwarrior access with workflow safety restrictions.
- `jacazul.taskwarrior.core`: project isolation, focus persistence, cache
  management, and task/broker boundaries.

Preserve these domain contracts unless a design decision explicitly changes
 them:

- Per-project state isolation through `PROJECT_ID`.
- Session-specific focus and cache state; never leak context across sessions.
- UUID-first task references; short UUIDs are presentation-only identifiers.
- Structured annotations such as `DECISION`, `RESEARCH`, `BLOCKED`, `LESSON`,
  `OUTCOME`, and `HANDOFF`.
- `OUTCOME` is required before a task can be closed.
- Errors, warnings, tips, and cache signals are operational interface output.
  Errors should explain the next corrective action.
- Normal output should remain quiet unless state changes, an error occurs, or
  debug mode is enabled.
- Taskwarrior compatibility is a migration constraint, not the final
  architecture. Keep persistence and workflow logic behind clean boundaries
  so a native store or future server can be introduced later.

## Repository Layout

```text
cmd/jaflow/       CLI executable and global option parsing
internal/cli/      command implementations and command routing
internal/config/   application-wide options and dispatch integration
internal/testharness/ isolated test fixtures and fake external commands
README.md         migration context and public design direction
go.mod            module and dependency declarations
```

The command model is intentionally Git-like:

```text
jaflow <command> [<args>]
```

The command registry should become the source of truth for command metadata,
routing, and help rendering. Keep parser details subordinate to the CLI
experience.

## CLI Reference Pattern

Use `~/source/candango/nvimim` as the reference for CLI composition and user
experience. Its implementation is a style reference, not a dependency or a
behavioral source for workflow semantics.

- Keep `cmd/jaflow/main.go` thin: construct global options, configure the
  parser, register commands, handle help/version, and map final errors to
  stderr and exit status.
- Put command behavior in `internal/cli`, with one concrete command type per
  command and an `Execute(args []string) error` method.
- Use a small options handoff boundary (`SetAppOptions` or an equivalent
  consumer-owned contract) instead of global mutable state.
- Register commands with a short summary and detailed help text. Positional
  argument validation belongs to the command that owns those arguments.
- Return errors from command code; do not call `os.Exit` deep inside command
  packages. The process boundary owns termination.
- Split workflow behavior by command. A command such as `focus`, `status`,
  `done`, `session`, or `cache` should have its own command type and
  `Execute(args []string) error` boundary, following the `nvimim` pattern.
- Do not translate the Python `FlowManager` into a new Go monolith. Shared
  behavior belongs in small, focused workflow, persistence, or cache
  components with clear responsibilities.
- Keep command complexity local and observable: each command gets focused
  contract tests, and shared components get their own behavior tests.
- Test command behavior directly using temporary directories, controlled stdin,
  local HTTP test servers, and filesystem assertions. Do not make unit tests
  depend on the user's home directory or live network services.
- Preserve readable, user-facing output for successful state changes while
  sending failures to stderr through the process boundary.

## Engineering Rules

- Write idiomatic Go: explicit control flow, small concrete types, and
  behavior-based interfaces only when a real consumer or test seam requires
  one.
- Prefer the standard library. Avoid framework or Java-style layering.
- Preserve line-of-sight readability: handle errors early and keep the happy
  path left-aligned.
- Wrap actionable underlying errors with `%w`; use `errors.Is` and
  `errors.As` for classification.
- Keep client launchers (Pi, Claude, Copilot, Gemini, and OpenCode), shell
  bootstrap, hatch/prompt generation, persona rendering, and GitHub broker
  concerns out of the core workflow domain.
- Do not introduce concurrency until the behavior needs it and ownership,
  cancellation, and exit paths are explicit.
- Never embed credentials, tokens, or secrets in source, fixtures, logs, or
  generated output. Treat external tickets and future server inputs as
  untrusted data.

## Validation

Before considering a Go change complete:

```bash
gofmt -w <touched-go-files>
goimports -w <touched-go-files>  # when installed
go test ./...
go vet ./...
```

The repository has no Makefile-specific test target yet. The conventional
race-enabled command for this project is:

```bash
go test -race -v ./...
```

Run it for behavioral changes; use `go test ./...` and `go vet ./...` as
focused local checks when appropriate. If `goimports` is unavailable, use
`gofmt` and report that import organization was skipped. Add focused tests for
new command behavior and regression tests for compatibility-sensitive workflow
semantics.

The supported Go versions are 1.25 and 1.26. Go versions older than 1.25
are EOL for this project. Do not claim a change is validated across both
supported versions unless the relevant runs were actually performed.

## Runtime Data and Sandbox Safety

The project checkout is the code workspace. User/project runtime data is the
protected original and must remain untouched by tests and development tools
unless explicitly requested.

Protect these sources by default:

- the user's real `TASKDATA` and Taskwarrior database;
- the user's real `JACAZUL_HOME`, cache, session, and focus files;
- the user's home configuration and credentials;
- live external services and production tickets.

Use this operating model:

- Code, tests, and documentation for the current feature belong in this
  checkout, under the task's scope.
- `[INVESTIGATE]` and `[REVIEW]` must not mutate user/project runtime data.
- `[EXECUTE]`, `[REFINE]`, and `[TEST]` use isolated temporary or persistent
  runtime state, never the real home, database, cache, or network.
- The persistent runtime sandbox root is:

  ```text
  ~/.jacazul-ai/sandboxes/<project-id>/<session-id>/
  ```

- The absolute sandbox path must be exposed in task/session context so agents
  can inspect it directly.
- Use `git worktree` or a clone only when the code itself needs a separate
  experimental checkout; it is not a replacement for isolated test data.
- Do not add a `.gitkeep` or ignored runtime-data directory to the project
  tree. Test fixtures and harness code belong in the repository; their data
  belongs in the sandbox or `t.TempDir()`.
- Before any operation against external services or real user data, require an
  explicit user request and verify the exact target.

A user request to implement a feature authorizes code and test changes in this
checkout; it does not authorize touching real runtime data, production
services, commits, merges, or pushes.

## Git and Workflow Safety

- Work from the current task/plan context before material repository changes.
- Keep persistent task and commit data in English; communicate with the user
  in the session language.
- Do not use raw Taskwarrior commands in Jacazul workflow operations. The
  compatibility layer belongs in the controlled workflow boundary.
- Stage only task-relevant files. Never use `git add .` or `git add -A`.
- Do not commit or push without explicit user confirmation.
- Use conventional commit messages, keep titles under 50 characters, and do
  not include internal task UUIDs or Copilot trailers.

## Source of Truth and Decisions

When behavior is unclear, inspect the corresponding workflow implementation
and tests in `jacazul-ai-cli` first, then record the compatibility decision in
this repository's documentation or design notes. Ignore hatch and client
prompt details unless they expose a workflow contract. Prefer a small
executable vertical slice over speculative abstractions.
