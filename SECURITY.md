# Security policy

NoBluff 0.4.x includes a Go detector, a shell/plugin launcher, a Node npm launcher and optional text skills. The detector reads local session files, makes no network requests and does not send transcripts. Findings can quote session answers; debug logging can write local metadata when explicitly enabled. Treat transcripts, findings and adapter state as private.

The optional Managed Agents example is a network service with separate anthropic/flask dependencies, API and webhook credentials and a private persistent SQLite state file. It is experimental. Its unit tests use doubles; they do not validate a real signed delivery or production hosting.

The detector is an availability-oriented heuristic. Hook read/parse errors allow the turn to finish with a warning when possible. One correction round then warning is documented behavior. Do not use zero findings as a security or correctness guarantee, or run destructive commands solely to obtain evidence.

Build source and binary digests are in bin/build-manifest.json. Rebuild with the pinned compiler and compare with python scripts/build.py --check. Package verification checks an exact tarball; a reproducible hash does not prove that source code is safe.

| Version | Status |
|---|---|
| 0.4.x | Supported corrective line; local 0.4.1 candidate requires release before registry installation |
| <= 0.3.x | Historical; known limitations documented in CHANGELOG, no new fixes planned |

Report vulnerabilities privately with GitHub security advisories: https://github.com/provenwork/nobluff/security/advisories/new . Do not attach private transcripts or credentials. Include a synthetic reproduction, affected version and expected/observed behavior. No response-time guarantee is claimed.