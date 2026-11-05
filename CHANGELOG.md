# Changelog

## 0.2.0 — 2026-11-05

- Renamed from Procureur to NoBluff, a name that says the promise in any language. The repository moved from `provenwork/procureur` to `provenwork/nobluff`; GitHub redirects the old URL.
- Install with `/plugin install nobluff@provenwork` (was `procureur@provenwork`) or `npx skills add provenwork/nobluff`.
- The umbrella skill is now `nobluff`. The five skills are unchanged.

## 0.1.0 — 2026-11-05

- Five skills: `evidence-first`, `no-placeholders`, `fact-gate`, `verdict-format`, `self-audit`, plus the `procureur` umbrella skill.
- Claude Code plugin manifest, installable from the `provenwork/skills` marketplace.
- `examples/double-charge`: a reproducible demo where the test suite is green while a retried webhook books 149.00 EUR twice.
- Tests for the skill pack structure and the demo, run in CI on Python 3.10 and 3.13.
