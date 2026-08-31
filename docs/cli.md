# Jaflow CLI Navigation

Jaflow's root help is organized by operator intent rather than alphabetic
command name. Use it to choose the workflow family first, then use
`jaflow help <command>` for the exact contract.

```bash
jaflow help
jaflow help status
jaflow help migrate
```

## Root help taxonomy

The root help exposes canonical commands once in this fixed order:

### Start and organize work

- `plan`: create an initiative and chained tasks;
- `roadmap`: manage strategic phases;
- `rename`: rename an initiative;
- `backlog`: pause an initiative;
- `activate`: restore a backlog initiative.

### Examine workflow state

- `help`: show the agent workflow briefing;
- `status`: inspect project task state;
- `ponder`: render the project dashboard;
- `plans`: list initiative summaries;
- `next`: list ready tasks;
- `tree`: inspect dependency markers;
- `active`, `blocked`, `overdue`: inspect derived task views.

### Work on the current task

- `focus`: inspect or switch the current anchor;
- `execute`: start ready work;
- `outcome`: record the completion result;
- `done`: complete a task after its outcome;
- `handoff`: transfer execution context;
- `reopen`: return completed work to pending;
- `discard`: archive work with an audit outcome.

### Change and reprioritize work

- `amend`: update task description or ticket metadata;
- `urgent`: raise priority and urgency;
- `block`: add a dependency;
- `unblock`: remove a dependency;
- `wait`: postpone readiness until a date.

### Preserve and maintain context

- `session`: inspect or resume session state;
- `note`: add or delete structured context;
- `notes`: list annotations;
- `context`: inspect direct and inherited context;
- `ticket`: link external ticket metadata.

### Maintain derived workflow state

- `cache`: inspect or clear derived output cache.

### Prepare and integrate changes

- `commit`: draft a conventional commit without staging or committing files.

### Migrate legacy state

- `migrate`: import an explicit Taskwarrior snapshot into native Jaflow.

## Compatibility aliases

Aliases remain routable for existing agents but are hidden from the root
canonical list:

| Alias | Canonical command |
|---|---|
| `initiative`, `ini` | `plan` |
| `inis`, `initiatives` | `plans` |
| `ship` | `roadmap ship` |

Detailed help remains available for aliases:

```bash
jaflow help ini
jaflow help initiatives
jaflow help ship
```

## Recommended navigation loop

Use the smallest command that answers the current workflow question:

```text
orient → inspect → focus → execute → outcome → done → next focus
```

Typical sequence:

```bash
jaflow status
jaflow next <initiative>
jaflow focus task <uuid>
jaflow execute <uuid>
jaflow outcome <uuid> "Describe the result"
jaflow done <uuid>
jaflow focus plan <initiative>
```

Do not execute a blocked task. `pending` is not equivalent to `ready`; native
readiness is determined from dependency state and wait dates.

## Output and error contract

Successful state changes go to stdout. Errors include an `ACTION:` prompt that
explains the next valid command. Healthy report commands may be quiet when no
state changed, while explicit report commands such as `status`, `ponder`, and
`cache info` render their state.

Use full UUIDs as identity and short UUIDs for display. The root help is a
navigation map; command-specific help is the detailed operational contract.
