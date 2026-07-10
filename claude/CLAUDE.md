## Picking the right models for workflows and subagents

Rankings, higher = better. Cost reflects what I actually pay (OpenAI has really generous limits), not list price. Intelligence is how hard a problem you can hand the model unsupervised. Taste covers UI/UX, code quality, API design, and copy.

| model       | cost | intelligence | taste  |
|-------------|------|--------------|--------|
| gpt-5.5     | 9    | 8            | 5      |
| gpt-5.6-sol | 8    | 9            | 7      |
| sonnet-5    | 5    | 5            | 7      |
| opus-4.8    | 4    | 7            | 8      |
| fable-5     | 2    | 10           | 9      |

How to apply:
- These are defaults, not limits. You have standing permission to override them: if a cheaper model's output doesn't meet the bar, rerun or redo the work with a smarter model without asking. Judge the output, not the price tag. Escalating costs less than shipping mediocre work.
- Cost is a tie-breaker only; when axes conflict for anything that ships, intelligence › taste > cost.
- Bulk/mechanical work (clear-spec implementation, data analysis, migrations): gpt-5.5 - it's effectively free.
- Hard, long-horizon work (ambiguous coding, research, science, computer use, cybersecurity): gpt-5.6-sol. Prefer it when the extra intelligence materially improves the outcome; keep gpt-5.5 for routine throughput.
- Anything user-facing (UI, copy, API design) needs taste ≥ 7.
- Reviews of plans/ implementations: fable-5 or opus-4.8, optionally gpt-5.6-sol as an extra high-intelligence perspective.
- Never use Haiku.
- Don't substitute gpt-5.6-terra or gpt-5.6-luna for gpt-5.5 by default: Terra is positioned as competitive with 5.5 rather than better, and Luna optimizes for speed/cost rather than capability.
- Mechanics: OpenAI models are only reachable through the Codex CLI - `codex exec` / `codex review` (my `~/-codex/config.toml` defaults to gpt-5.5). Use `-m gpt-5.6-sol` for Sol; it requires Codex CLI 0.144.0 or newer. Use the codex-implementation, codex-review, and codex-computer-use skills; for work they don't cover (investigation, data analysis), run `codex exec -s read-only` directly with a self-contained prompt.
- Claude models (sonnet-5, opus-4.8, fable-5) run via the Agent/Workflow model parameter.
Using OpenAI models inside workflows and subagents (the model parameter only takes Claude models, so use a wrapper):
- Spawn a thin Claude wrapper agent with `model: 'sonnet'`, `effort: 'low'` whose prompt instructs it to write a self-contained Codex prompt, run `codex exec` via Bash (adding `-m gpt-5.6-sol` when Sol is wanted), and return the result.
