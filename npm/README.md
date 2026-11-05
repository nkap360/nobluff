# nobluff

**Did your coding agent say "tests pass" without running them?** Find out in one command:

```bash
npx nobluff scan
```

```text
NoBluff scanned 50 Claude Code sessions (412 turns, 230 final answers).
3 answers claimed something the transcript contradicts, in 2 sessions.

  tests-not-run      2
  tests-failed       1
```

It reads your local Claude Code transcripts (`~/.claude/projects`), turn by turn, and flags answers that claim something the session itself contradicts:

| Rule | The answer says… | …but the transcript shows |
|---|---|---|
| `tests-not-run` | tests pass | no test command after the last file change |
| `tests-failed` | tests pass / fixed / works | the last test run failed |
| `tests-skipped` | all tests pass | tests were skipped and the answer never says so |
| `unverified-claim` | verified / works / fixed | code was edited and nothing ran since |

Local only: no model, no network, nothing leaves your machine.

```bash
npx nobluff last                 # latest session of the current folder
npx nobluff scan -n 200          # more history
npx nobluff check session.jsonl  # one transcript, JSON output
```

## Stop the bluff before it reaches you

The same detector runs as a Claude Code Stop hook: when the agent tries to finish on a claim its own session contradicts, it is sent back to prove it or retract it.

```text
/plugin marketplace add provenwork/skills
/plugin install nobluff@provenwork
```

Source, rules, tests and limitations: https://github.com/provenwork/nobluff
