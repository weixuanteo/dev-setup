---
name: codex-review
description: Review repository changes through supervised Codex. Use when another skill requests Codex review of uncommitted changes, a branch diff, or a commit.
---

# Codex Review

Set `RUNNER` to the absolute path of `../codex-implementation/scripts/codex-run.mjs` relative to this skill.

## 1. Review

Start with exactly one target:

```bash
node "$RUNNER" review --cwd <working-directory> <target>
```

Replace `<target>` with `--uncommitted`, `--base <branch>`, or `--commit <sha>`. Poll `node "$RUNNER" wait <run-id>` until terminal, then read `node "$RUNNER" result <run-id>`.

Proceed when Codex exits successfully. Never pass a prompt or resume a review.

## 2. Disposition

Verify every finding against the code. Send accepted findings to `codex-implementation`; record dismissals with evidence. After fixes pass verification, start a fresh review.

Finish when every finding has a disposition and a fresh review has no actionable findings. If Codex produced the change, do not count Codex review as independent.
