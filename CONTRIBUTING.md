# Contributing

NoBluff asks agents for evidence. Contributions follow the same rule.

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

Python tests use the standard library (3.10+). The detector uses Go's standard library.
Before changing the detector, run `go vet ./...` and `go test -count=1 ./...` from `detector/`.
The public synthetic regression protocol is `go test -count=1 -run TestPublicBenchmark -v ./...`.
It is chosen around known defects and does not estimate accuracy on real sessions.

Build with pinned Go1.27.3: `python scripts/build.py`, then verify with
`python scripts/build.py --check`. The second command rebuilds all six targets and
compares bytes and manifest values. Never hand-replace only one platform binary.
`python scripts/verify_package.py` checks and exercises the actual local tarball,
without publishing or installing dependencies. Cross-builds do not test every OS at runtime.
Release preparation must keep source, plugin, npm and build-manifest versions equal.
The candidate workflow uploads verified artifacts only; registry publication requires a
separate configured trusted-publisher environment and explicit release authorization.
Never add private session recordings to test data: minimize a defect to a synthetic fixture.

## Conduct

Be precise and be kind. Criticise claims, not people. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
