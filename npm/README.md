# nobluff

Review selected contradictions between Claude Code final claims and visible commands. This deterministic local heuristic cannot prove code correctness or agent honesty.

Local corrective candidate: **0.4.1**, not published by this preparation. After release:

```bash
npx nobluff@0.4.1 demo
# synthetic example; expected exit 1, tests-not-run
```

The package includes a synthetic fixture under examples/ and source/binary SHA256 metadata in build-manifest.json. Optional scan/last read private local sessions; they do not transmit them. npm may contact the registry to obtain this package. Detector exit codes:0 no finding,1 findings,2 operational error. Scan distinguishes answers from rule findings and reports parsing gaps.

No global install is needed for npx; npm may retain a cache. If globally installed, remove with npm uninstall -g nobluff. The package does not install a plugin hook automatically. Claude Code plugin removal and limitations: https://github.com/provenwork/nobluff#install-update-disable-remove

Known limits include dynamic shells, custom runners, unusual claim wording, irrelevant or fabricated checks and false positives. Full source, synthetic regression protocol, build verification, plugin setup and optional experimental webhook adapter: https://github.com/provenwork/nobluff