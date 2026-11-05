# Procureur

[![ci](https://github.com/provenwork/procureur/actions/workflows/ci.yml/badge.svg)](https://github.com/provenwork/procureur/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/provenwork/procureur?sort=semver)](https://github.com/provenwork/procureur/releases)
[![license: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Agent Skills](https://img.shields.io/badge/format-Agent%20Skills-6f42c1.svg)](https://github.com/provenwork/skills)
[![Claude Code plugin](https://img.shields.io/badge/Claude%20Code-plugin-d97757.svg)](#install)

**Make your coding agent prove its work.** Procureur is a skill pack that turns Claude Code, Codex or Cursor into the prosecutor of its own output: no placeholders, no "optionally", no "should work". Evidence before claims, decisions instead of options, a verdict you can act on.

## Install

```bash
# Claude Code
/plugin marketplace add provenwork/skills
/plugin install procureur@provenwork

# Any agent that reads SKILL.md (Claude Code, Codex, Cursor, OpenClaw…)
npx skills add provenwork/procureur
```

Two lines in Claude Code, one command anywhere else. Five skills. No dependencies.

## Demo: the green test that charges twice

![The shipped test passes; replaying the webhook books 149.00 EUR twice](assets/demo.svg)

A payment webhook handler ships with a passing test. Payment providers retry webhooks, so the same event can arrive twice. Each delivery is booked, and one 149.00 EUR payment ends up as 298.00 EUR in the ledger. Reproduce it in 30 seconds with Python 3.10+ and nothing else, from [`examples/double-charge`](examples/double-charge) in a copy of this repository:

```bash
cd examples/double-charge
python -m unittest test_payments   # OK: the suite is green
python replay_check.py             # FAIL: 2 ledger entries, 298.00 EUR
python replay_check.py --fixed     # PASS: 1 ledger entry, 149.00 EUR
```

"Tests pass" was true and the bug was still there. `evidence-first` makes the agent test the business rule (one payment, one entry) against the real-world failure, instead of stopping at the existing suite. CI runs these three commands on every push, so the demo cannot silently rot.

## How it works

Each skill is a short `SKILL.md` your agent loads. Together they change what the agent is allowed to hand you:

| Skill | What it forbids | What it demands |
|---|---|---|
| `evidence-first` | "should work", "probably", untested claims | run it, cite `file:line`, show the output |
| `no-placeholders` | TODO, "insert here", "optionally", three alternatives | one complete deliverable, decisions stated |
| `fact-gate` | editing a file blind | callers, schemas and the user's instruction stated before the edit |
| `verdict-format` | burying the answer under caveats | bottom line first, proof second, residual risk last |
| `self-audit` | silent failures, swallowed errors | a closing list: verified / assumed / not checked |

Read them, edit them, delete the ones you disagree with.

## Example: the contract in one answer

An illustrative answer to "fix the double charge", in the shape Procureur asks for:

> **Fixed.** `handle_payment_succeeded` now records the provider `event_id` in `receipts` and books the ledger entry in the same transaction (`fixed/payments.py:21`).
> **Proof:** `python replay_check.py --fixed` gives 1 ledger entry, 149.00 EUR, after 2 deliveries (and after 5, in `tests/test_double_charge.py`).
> **Not checked:** concurrent deliveries across processes. SQLite serialises writes here; on Postgres, the unique key on `event_id` is what makes this safe.

## Why

Eight years shipping GenAI in a Swiss private bank taught me one thing about AI output: the expensive failures are never the loud ones. They are the fluent, confident, unverified ones. Procureur is the working style I impose on every agent I use, written down so you can impose it too.

## Limitations

- It does not make the model smarter. It changes what the model is allowed to hand you.
- It costs tokens: evidence means running things. Expect longer turns on small tasks and fewer round trips on real ones.
- It is opinionated. If you like being offered three options, this is not for you.
- No controlled with/without comparison has been published yet. The demo shows the failure class Procureur targets, not a measured effect.

## Contributing

Bug reports and skill proposals are welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md); agents working on this repo should read [AGENTS.md](AGENTS.md). Changes are listed in [CHANGELOG.md](CHANGELOG.md), and security reports go through [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE).
