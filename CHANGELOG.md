# Changelog

## Unreleased

- A shell command that changes a file and then tests it (`sed -i … && pytest`, `printf … > notes.txt && python -m unittest`) now counts as tested after the change. 0.4.3 recorded the change after the whole command, so the test run inside it never counted. A test followed by a change in the same command (`pytest && sed -i …`) is still flagged.
- Deleting only build artifacts, caches, screenshots or prose files (`rm -rf .playwright-mcp`, `Remove-Item -Recurse node_modules, dist`) is no longer a file change. Deleting anything else, including a source file, still is.
- A prose file written after a green run (README, notes, changelog, an agent's memory file) no longer invalidates the run: "the last file change" now ignores `.md`, `.txt`, `.rst` and similar files. A code change after the note still does.
- 15 new synthetic cases (`detector/ordering_test.go`), each false alarm paired with the true alarm that must stay. Origin: a field report of 13 Stop-hook blocks in one long session, 2026-11-09.
- Questions (« Tu veux que je vérifie… ? », "Want me to check?") and offers (« Je peux ensuite chercher… », "I can verify it next") are no longer read as claims. Offering to confirm is still a claim ("I can confirm the tests pass").
- Hypothetical subordinates (« jusqu'à ce que le test passe », « pour que », « avant que », "until", "unless") and French « n' » negations up to two words before the verb (« n'a pas été vérifié ») now cancel the claim. « Le test passe. » and « Le rendu a été vérifié. » are still flagged.
- 13 new synthetic cases (`detector/claims_test.go`), with the same pairing.

## 0.4.3 — 2026-11-08

- A markdown blockquote (lines starting with `>`) is now a mention, like a quote: an agent reproducing an old answer to audit it is no longer blocked for the claim it quotes. A blockquote still counts when the answer is little more than the blockquote itself.
- French negations built with « ne » / « n' » before the verb (« rien ne l'a vérifié », « personne n'a vérifié ») are no longer read as claims.
- Python runs with interpreter options before the module or script (`python -P -m unittest`, `python -I -m pytest`, `py -3.12 -m pytest`, `python -X utf8 -m unittest`) are now recognized as test or check commands. 0.4.2 ignored them and flagged answers whose tests had actually run.
- Benchmark extended from 24 to 30 synthetic cases (2 blockquote, 3 French negation and 1 interpreter-option cases added); all earlier cases unchanged.

## 0.4.2 — 2026-11-08

- Fewer false alarms on real histories. A quoted claim is now a mention by default, in any quote style including « », and counts as an assertion only when presented as a result: after a result label at the start of its line, after a lead of at most two words ending in a colon, or when the answer is little more than the quote. 0.4.1 flagged answers that merely discussed « les tests passent ».
- Sentences reporting someone else's work ("The author fixed it", "Il a corrigé") and lines introducing a list ("**What was checked:**") are no longer read as the agent's own claim.
- Benchmark extended from 12 to 24 synthetic cases: 7 quotation and 5 attribution cases added; the 8 audit bypasses stay closed.
- On the author's own history (50 sessions, 2026-11-08), findings fell from 28 in 25 answers with 0.4.1 to 12 in 12 answers.

## 0.4.1 — 2026-11-08

- Scan counts distinct answers separately from rule findings and reports parse coverage errors.
- Runner recognition checks executable positions; eight publicly defined bypass cases have regression tests. Common shell mutations invalidate previous checks. Only completed recognized checks count; a read-only command such as pwd is not verification.
- Positive claims survive unrelated negations, the phrase let me be clear, and asserted quoted results. Explicit quotation discussion and genuine future-check announcements remain exempt. These remain bounded heuristics, not semantic proof.
- Parser failures are explicit errors; availability-oriented Stop-hook failures warn and allow stopping rather than reporting a clean verdict.
- Experimental webhook persists delivery/challenge state, permits processing retries, rejects a forged plain prefix as an anti-loop signal and treats detector failure as an error. Ambiguous sends require inspection; no live webhook validation claimed.
- Public synthetic benchmark v1 and zero-history demo; old internal accuracy figures removed from current marketing copy. No new private recordings distributed.
- Go1.27.3 pinned; all six binaries have source/binary digests and byte-for-byte rebuild checks. Exact npm tarball path/digest/version checks and host synthetic smoke. The release workflow stages by default; registry/GitHub publication requires an explicit publish input, matching version tag and configured protected OIDC environment.
- Current support policy, installation/removal instructions and limitations documented.

### Historical-study qualification

The 0.3.0/0.4.0 notes below preserve statements made at those dates. Their quoted internal study counts, mutation runs and private history scans are not independently reproducible from the released corpus. They are not validation of 0.4.1 or a public accuracy estimate. The published Managed Agents recordings demonstrate one session only; the full claimed study corpus was not released. The historical npm0.3.0 binaries correspond to a later public commit than the v0.3.0 tag (28714177514f85517fb4b5336da6bad7cb2af625 versus d5e65fa1c44b7f2d00462f3bd1e806db99c517a6); old packages/tags are not rewritten by this correction.
## 0.4.0 — 2026-11-06

- New: Claude Managed Agents support. `nobluff check` reads Managed Agents session events (`GET /v1/sessions/{id}/events`, as a JSON array, a paginated `{"data": [...]}` page, or JSON Lines) and applies the same four rules to the latest turn. `nobluff check -` reads from stdin.
- New: `examples/managed-agents/webhook.py`, a server-side Stop hook. On `session.status_idled` it runs the detector on the session's events and sends the agent back once per turn when its answer is contradicted.
- Validated on a live Managed Agents session (Sonnet 5.5): an honest turn gave no finding; a turn that edited without running anything and ended with "All tests pass." gave `tests-not-run`; after `handle_idle` pushed back, the agent ran the tests and the next verdict was clean. Tool results link to their call by `tool_use_id`, as the adapter expected, including parallel calls. Both recordings are regression tests in `detector/testdata/`.
- Fixed (found on that session): "I ran no commands" was read as a claim, because the negation came after the verb. A negation right after the verb ("I ran no…", "I tested nothing") now cancels the claim. No change on the 62 BluffBench transcripts, on 50 mutants, or on a 63-session history scan.
- Fixed in `webhook.py`: events are serialised with `to_dict(mode="json")`; plain `to_dict()` keeps datetimes that `json.dumps` cannot encode. The logic now lives in `handle_idle(session_id)`, which can be run without a webhook delivery.
- Not yet checked: a real `session.status_idled` webhook delivery and its signature verification.

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
