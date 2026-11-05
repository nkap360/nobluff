# AGENTS.md

Instructions for coding agents working **on this repository**. To use NoBluff in your own project, install it instead (see README).

- This repository practises what it ships: apply `skills/evidence-first`, `skills/fact-gate` and `skills/self-audit` to every change here.
- Run `python -m unittest discover -s tests -v` before claiming a change works, and paste the result.
- Standard library only, Python 3.10+. Do not add dependencies.
- Keep each `SKILL.md` under about 60 lines, with YAML frontmatter `name` (equal to the folder name) and `description`.
- A change to a skill's behaviour needs an entry in `CHANGELOG.md` and, when possible, an example that demonstrates it.
- `examples/double-charge/payments.py` is buggy on purpose. Do not "fix" it; the fix lives in `examples/double-charge/fixed/`.
