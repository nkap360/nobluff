# NoBluff

**Find out how often your coding agent said "tests pass" when nothing ran, and stop it from happening again.**

[![One-minute demo: your agent says "all tests pass", the transcript says otherwise, NoBluff flags it](assets/demo-poster.jpg)](assets/demo.mp4)

<p align="center"><sub>One minute, with sound. Click the image to play.</sub></p>

```bash
npx nobluff scan
```

```text
NoBluff scanned 50 Claude Code sessions (265 turns, 165 final answers).
12 answers had 12 findings to review, in 3 sessions.

  unverified-claim   9
  tests-not-run      3
```

That was the author's own history on 2026-11-08, with 0.4.2. Each finding is a sentence to review, not a verdict: some are claims the agent never backed with a command, a few describe what a tool does. One command, read-only: nothing leaves your machine. Your number will differ. No history yet? `npx nobluff demo` runs on a synthetic session.

## Then make it impossible

Install the Stop hook in Claude Code:

```text
/plugin marketplace add provenwork/skills
/plugin install nobluff@provenwork
```

When the agent ends its turn with a claim its own transcript contradicts, the turn is blocked:

```text
Agent:   [edits payments.py, runs nothing] ... All tests pass.
NoBluff: You say "All tests pass.", but no test command ran after your last file change.
         Run the tests and show the result, or remove the claim.
```

The agent runs the check or takes the claim back. NoBluff blocks once, then only warns, so it cannot trap the agent in a loop. One small Go binary: no model, no network, the same verdict every time.

## How it works

| Rule | Recognized claim | Transcript condition |
|---|---|---|
| `tests-not-run` | tests pass | no recognized test invocation after the last recognized edit |
| `tests-failed` | tests pass / fixed / works | the last recognized test invocation failed |
| `tests-skipped` | all tests pass | the last run reports skipped tests, without an explicit test-skipping disclosure |
| `unverified-claim` | verified / works / fixed | code edited without a recognized check since |

Executable names are checked at command positions; `echo pytest` does not count as pytest. Some direct shell mutations, such as `sed -i`, invalidate earlier runs; a change followed by a test in the same command (`sed -i … && pytest`) counts as tested. Deleting only build artifacts, caches or screenshots, and writing prose files (README, notes), do not invalidate earlier runs. This is a bounded shell parser, not a shell interpreter: wrappers, aliases, scripts that edit files, dynamic commands, other languages and unusual wording can be missed. Any recognized check can still be irrelevant or fabricated. A quoted claim is treated as a mention unless it is presented as a result (after a label such as "Result:", after a very short lead ending in a colon, or as almost the whole answer); announced future checks, questions, offers ("I can verify it next"), hypothetical clauses ("until the tests pass") and reports about someone else's work are exempted. Checks done by reading a web page are not recognized as verification. False positives remain possible.

Exit codes for `check` and `demo`: **0** no finding, **1** findings to review, **2** input/operational error. Zero findings is not a certificate. `scan` reports distinct answers separately from rule findings and identifies parse coverage gaps. Its default is the 50 most recent top-level sessions; subagent folders are excluded. `last` falls back to the latest session of any project if this folder has none.

Before launch, a competing model audited NoBluff adversarially. The eight bypasses it found, such as `echo pytest` counting as a test run, are fixed and now part of the public regression benchmark (see Public, reproducible checks).

## Limits

NoBluff is a local, deterministic heuristic for Claude Code transcripts. It can flag a recognized claim such as “All tests pass” when no recognized test command followed the last recognized edit. It cannot prove that code is correct, that a command really tests the change, or that an agent is honest. Other agents can use the optional text skills; the executable Stop hook is for Claude Code. Zero findings is not a certificate, and false positives remain possible.

## Install, update, disable, remove

Claude Code marketplace installation:

```text
/plugin marketplace add provenwork/skills
/plugin install nobluff@provenwork
```

The plugin invokes the platform binary through its Stop hook. It blocks once, then warns on the correction round to avoid a loop. `NOBLUFF_MODE=warn` requests warnings only. Input/internal errors allow the turn to finish and emit a warning when possible; this availability choice is not a clean verdict.

Update through Claude Code's plugin manager and verify the selected version against its manifest. Disable or uninstall `nobluff` through the same plugin manager; verify the NoBluff Stop hook is absent before trusting that it is disabled. A manually copied hook must be removed from the settings file where it was added.

For one-shot CLI use, `npx nobluff demo` does not create a global installation. npm may retain a cache; do not delete unrelated caches as an uninstall step. If installed globally, remove only this package with `npm uninstall -g nobluff`. For skills-only installs, use the skill installer's removal command or remove only the installed NoBluff skill directories; these text skills do not install a detector hook in other agents.

History review:

```bash
npx nobluff scan   # the 50 most recent sessions
npx nobluff last   # the latest session of this folder
```

These commands read local transcripts. No transcript is sent by the detector. Review output can quote final answers: keep it private when it contains private material. `NOBLUFF_DEBUG`, if configured, writes local diagnostic metadata to the named file.

## Public, reproducible checks

```bash
python -m unittest discover -s tests -v
cd detector
# synthetic benchmark: complete constructions and expectations are in regressions_test.go
go test -count=1 -run TestPublicBenchmark -v ./...
go vet ./...
go test -count=1 ./...
cd ..
python scripts/build.py --check
python scripts/verify_package.py
```

The benchmark contains **30 explicitly constructed cases: eight known bypass regressions, four comparison cases, nine quotation cases, five attribution cases, three French negation cases and one interpreter-option case** (definition dated 2026-11-08). It is a regression corpus chosen around known defects, not a random sample and not a real-world accuracy estimate. Results must be recalculated with the commands above. No claim about user adoption, long-session bluff rates, or skill efficacy follows from it. Earlier releases quoted internal studies whose full protocols and corpora were not public; their historical notes remain labelled as such, and those numbers are not evidence for this candidate.

Builds use pinned **Go 1.27.3**, `CGO_ENABLED=0`, `-buildvcs=false -trimpath -ldflags="-s -w"`. `bin/build-manifest.json` records source and binary SHA256 values. `build.py --check` actually rebuilds all six binaries and compares bytes; `verify_package.py` enumerates and hashes the exact tarball, then executes its host binary on a synthetic fixture. Cross-compilation does not establish runtime compatibility on every target. There are no detector runtime dependencies; the npm launcher requires Node 16 or newer. Build tooling requires Python 3.10+ and the pinned Go compiler; npm packaging requires npm and Node.

## Payment replay example

`examples/double-charge` deliberately contains a handler whose one happy-path test misses duplicate delivery. Run from that directory:

```bash
python -m unittest test_payments
python replay_check.py
# Expected exit 1: two ledger entries, 298.00 EUR for two deliveries of 149.00 EUR
python replay_check.py --fixed
# Expected exit 0: one entry, 149.00 EUR
```

This demonstrates a test-coverage gap, not a measured effect of installing text skills. Concurrent deliveries across processes are outside this demonstration.

## Managed Agents adapter — experimental

`examples/managed-agents/webhook.py` requires separately installed `anthropic` and `flask`, API credentials and webhook signing configuration. Set `NOBLUFF_BIN` to a verified local binary and `NOBLUFF_STATE` to a private writable SQLite file. It never downloads a detector automatically.

The adapter persists delivery/challenge state. Failed processing returns 503 and permits retry; a completed duplicate returns 204. An untrusted `[NoBluff]` prefix alone cannot suppress the challenge. Detector failure is an error, never a clean verdict. An ambiguous outbound send is not resent automatically: inspect the session events and reconcile its stored challenge before retrying. Run one server process; keep its database private and backed up. Offline SDK/Flask doubles cover control flow only. A real signed webhook, service deployment and end-to-end Managed Agents delivery are **not verified** by this candidate.

## Release workflow and alternatives

`release.yml` stages a verified candidate by default. Its optional publishing job requires the explicit `publish` input, a tag matching the package version, the GitHub `release` environment and a configured npm trusted publisher for this repository/workflow/environment. The workflow uses Node 24 and npm 11.6.2; this build requirement is separate from the launcher runtime requirement. It publishes the tested tarball and checks its digest before npm. Versions 0.4.1, 0.4.2 and 0.4.3 were published to npm manually on 2026-11-08, from this repository, before the trusted publisher for this workflow was configured.

For a text-only verification discipline, compare [superpowers verification-before-completion](https://github.com/obra/superpowers/tree/main/skills/verification-before-completion). [proof-of-done](https://github.com/B0yko/proof-of-done) also checks transcript evidence and offers a history audit. [verify-before-stop](https://github.com/ianymu/claude-verify-before-stop) checks a verification log. These are alternative mechanisms; no comparative performance superiority is claimed here.

## Skills and contribution

The five short instruction files are in `skills/`: evidence-first, fact-gate, no-placeholders, verdict-format and self-audit, plus the umbrella NoBluff skill. Their wording is an instruction contract; it does not enforce truth by itself. Read and edit them before use.

See [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), [CHANGELOG.md](CHANGELOG.md) and the [MIT license](LICENSE). This project was formerly named Procureur.