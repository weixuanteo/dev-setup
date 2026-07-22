---
name: codex-implementation
description: Delegate repository implementation to Codex through a supervised run. Use when another skill routes a mutating coding assignment to Codex.
---

# Codex Implementation

Set `RUNNER` to the absolute path of `scripts/codex-run.mjs` beside this file.

## 1. Frame

Write the assignment to a temporary file. Include the working directory, deliverable, allowed mutations, constraints, and verification. Forbid commits unless requested.

Proceed when the assignment is self-contained and checkable.

## 2. Supervise

Start one writer:

```bash
node "$RUNNER" start --cwd <working-directory> --prompt <assignment-file>
```

Poll the returned run id until terminal; each call waits at most 45 seconds:

```bash
node "$RUNNER" wait <run-id>
```

On failure, inspect `node "$RUNNER" result <run-id>`. Resume only a recoverable run, using evidence in a new prompt file:

```bash
node "$RUNNER" resume <run-id> --prompt <follow-up-file>
```

Proceed when Codex exits successfully. Never replace the runner with direct `codex` commands.

## 3. Verify

Inspect the diff and run the framed verification outside Codex. Resume with exact failures until both pass.

Finish when every changed file is in scope and every required check passes, or return the runner's explicit failure.
