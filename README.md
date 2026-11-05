# Procureur

**Make your coding agent prove its work.** Procureur is a skill pack that turns Claude Code, Codex or Cursor into the prosecutor of its own output: no placeholders, no "optionally", no "should work" — evidence before claims, decisions instead of options, a verdict you can act on.

```bash
# Claude Code
/plugin marketplace add provenwork/skills
/plugin install procureur@provenwork

# Any agent that reads SKILL.md (Claude Code, Codex, Cursor, OpenClaw…)
npx skills add provenwork/procureur
```

Two lines in Claude Code, one command anywhere else. Five skills. No dependencies.

## Sixty seconds

Ask your agent for a database migration **without** Procureur:

> "Here's a migration that should work. You may want to add an index later. Let me know if you'd like me to write tests!"

The same request **with** Procureur:

> "Migration written and executed against a scratch copy (`migrations/0007_add_index.sql`, 42 ms). Index added now, not later — the query in `api/orders.py:118` is the only caller and it filters on `customer_id`. Regression test added and passing (`tests/test_orders.py::test_index_used`, 3 cases). Not verified: production data volume; the estimate assumes < 10M rows."

Same model. Different contract.

## The five skills

| Skill | What it forbids | What it demands |
|---|---|---|
| `evidence-first` | "should work", "probably", untested claims | run it, cite `file:line`, show the output |
| `no-placeholders` | TODO, "insert here", "optionally", three alternatives | one complete deliverable, decisions stated |
| `fact-gate` | editing a file blind | callers, schemas and the user's instruction stated before the edit |
| `verdict-format` | burying the answer under caveats | bottom line first, proof second, residual risk last |
| `self-audit` | silent failures, swallowed errors | a closing list: verified / assumed / not checked |

Each skill is a short `SKILL.md`; read them, edit them, delete the ones you disagree with.

## What Procureur does not do

- It does not make the model smarter. It changes what the model is allowed to hand you.
- It costs tokens: evidence means running things. Expect 10–30 % longer turns on small tasks, and far fewer round trips on real ones.
- It is opinionated. If you like being offered three options, this is not for you.

## Why

Eight years shipping GenAI in a Swiss private bank taught me one thing about AI output: the expensive failures are never the loud ones. They are the fluent, confident, unverified ones. Procureur is the working style I impose on every agent I use, written down so you can impose it too.

MIT.
