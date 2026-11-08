# nobluff

Find out how often your coding agent said "tests pass" when nothing ran, and stop it from happening again.

```bash
npx nobluff scan   # read-only review of your 50 most recent Claude Code sessions
npx nobluff demo   # synthetic example, no history needed
```

To block these claims as they happen in Claude Code: `/plugin marketplace add provenwork/skills`, then `/plugin install nobluff@provenwork`. The hook blocks once, then only warns, so it cannot trap the agent in a loop. This package does not install the hook by itself.

Local and deterministic: no model, no network. Sessions are read on your machine and never transmitted; npm may contact its registry to fetch the package. Exit codes: 0 no finding, 1 findings to review, 2 operational error. Zero findings is not a certificate.

Limits: dynamic shells, custom runners, unusual claim wording, irrelevant or fabricated checks, and false positives. If installed globally, remove with `npm uninstall -g nobluff`. Source, regression benchmark, build verification and plugin removal: https://github.com/provenwork/nobluff
