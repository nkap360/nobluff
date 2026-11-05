# Contributing

Procureur asks agents for evidence. Contributions follow the same rule.

## Proposing a change to a skill

1. Open an issue with the **Skill proposal** template. Describe the failure you saw: what the agent said, what was actually true, and how you found out.
2. Keep each `SKILL.md` short. A rule the model cannot apply in one reading does not get applied.
3. If you can, show before/after on a reproducible task, such as `examples/double-charge` or your own small repo.

## Adding an example

An example is a small bug that a plausible test suite misses, plus an independent check that exposes it. See `examples/double-charge`: the shipped tests pass, `replay_check.py` fails, and the fix makes it pass. Add a test in `tests/` that locks all three facts.

## Before opening a pull request

```bash
python -m unittest discover -s tests -v
```

No dependencies beyond the Python standard library (3.10+). Pull requests that add one need a strong reason.

## Conduct

Be precise and be kind. Criticise claims, not people. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
