# Changelog

## 0.3.0 — 2026-11-06

- New: the NoBluff detector, a Claude Code Stop hook (Go, one static binary per platform, no dependency, no model, no network). When the agent tries to end its turn, it compares the final answer with what the session actually did, and blocks claims the transcript contradicts:
  - `tests-not-run`: "tests pass" with no test command after the last file change;
  - `tests-failed`: success claimed while the last test run failed;
  - `tests-skipped`: "all tests pass" while the run skipped tests the answer never mentions;
  - `unverified-claim`: "verified / works / fixed" after editing code, with nothing executed since.
  It blocks once, then only warns, so it can never trap the agent in a loop. `NOBLUFF_MODE=warn` reports without blocking.
- Why: BluffBench v1 (60 sessions) showed the agent never opened the NoBluff skills on its own. Instructions the model does not read cannot change its behaviour; the hook runs every time.
- Checked against recorded sessions: 0 alerts on 62 honest BluffBench transcripts, 32 of 32 caught when their test runs are removed but the "tests pass" claim is kept, and one live session where the hook blocked "All tests pass." and the agent retracted it.

## 0.2.1 — 2026-11-05

- Fixed: the `nobluff` umbrella skill (and the root `SKILL.md`) had an unquoted description containing ": ", which is invalid YAML. Claude Code registered it as a slash command but not as a skill the model can invoke. Found by BluffBench's isolation check.
- The pack test now rejects unquoted frontmatter values that a real YAML parser would reject, so this cannot ship again.

## 0.2.0 — 2026-11-05

- Renamed from Procureur to NoBluff, a name that says the promise in any language. The repository moved from `provenwork/procureur` to `provenwork/nobluff`; GitHub redirects the old URL.
- Install with `/plugin install nobluff@provenwork` (was `procureur@provenwork`) or `npx skills add provenwork/nobluff`.
- The umbrella skill is now `nobluff`. The five skills are unchanged.

## 0.1.0 — 2026-11-05

- Five skills: `evidence-first`, `no-placeholders`, `fact-gate`, `verdict-format`, `self-audit`, plus the `procureur` umbrella skill.
- Claude Code plugin manifest, installable from the `provenwork/skills` marketplace.
- `examples/double-charge`: a reproducible demo where the test suite is green while a retried webhook books 149.00 EUR twice.
- Tests for the skill pack structure and the demo, run in CI on Python 3.10 and 3.13.
