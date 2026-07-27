---
name: orchestrate
description: Delegate and route work across Claude and Codex subagents. Invoke BEFORE the Agent or Workflow tools whenever spawning subagents — even when no model choice seems needed, this skill decides the model and effort per assignment. Use when the user says orchestrate, act as the orchestrator, spin up / fan out subagents, delegate, parallelize across agents, use subagents, choose a model, or asks for an independent review.
---

## 1. Frame the assignment

Define each assignment's deliverable, working directory, mutation permissions, constraints, and verification. Keep tightly coupled reasoning together; split independent work into non-overlapping concurrent assignments.

## 2. Route the work

Rankings are personal defaults; higher is better. Cost reflects effective subscription cost, intelligence is the difficulty a model can handle unsupervised, and taste covers UI/UX, code quality, API design, and copy.

| model | cost | intelligence | taste |
|---|---:|---:|---:|
| `gpt-5.6-sol` | 8 | 9 | 7 |
| `opus-5` | 4 | 8 | 8 |
| `fable-5` | 2 | 10 | 9 |

- Route coding, research, science, computer use, cybersecurity, data analysis, and other long-horizon work to `gpt-5.6-sol`.
- Route long-horizon coding and agentic work that runs natively through the Agent or Workflow tools — especially subagent-heavy fan-outs and writer-verifier workflows — to `opus-5`; it needs no Codex bridge and coordinates parallel subagents reliably.
- Route plan or implementation review to `fable-5`; add `opus-5` or `gpt-5.6-sol` when another independent perspective is valuable. A reviewer must not be the model that produced the work.
- Never use Haiku.

This step is complete when every assignment is routed.

## 3. Dispatch

Run Claude models through the Agent or Workflow model parameter.

OpenAI models are reachable through Codex CLI, not the Agent model parameter. For delegated OpenAI work, spawn a Claude bridge using `model: opus` and `effort: low` with this contract:

> Act only as a Codex bridge. Invoke the routed Codex skill or command with the framed assignment, then return its result or failure. Do not perform the assignment yourself.

- `gpt-5.6-sol` is the default in `~/.codex/config.toml`; omit the model option.
- Use `codex-implementation` for implementation and `codex-review` for code review.
- Use `codex exec -C <working-directory> -s read-only -` for investigation, analysis, and other non-mutating work.
- Use `danger-full-access` only when the assignment explicitly requires access beyond the workspace and that access is authorized.

This step is complete when each worker has received a self-contained assignment and returned a result or explicit failure.

## 4. Gate the result

Inspect the returned artifact, diff, evidence, and verification output against the original deliverable and constraints; never accept a worker's summary as proof.

If the result misses the bar, refine the assignment and retry. Escalate to a smarter or more tasteful model when enough to recover; otherwise redo the work directly. Do not let multiple writers repair the same files concurrently.

Finish only when every delegated assignment is verified and either accepted, successfully retried, or explicitly redone.
