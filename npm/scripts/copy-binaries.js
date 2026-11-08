// Copies the six detector binaries from ../bin into the package before `npm pack` / `npm publish`.
"use strict";
const fs = require("fs");
const path = require("path");
const src = path.join(__dirname, "..", "..", "bin");
const dst = path.join(__dirname, "..", "bin");
const names = ["linux-amd64", "linux-arm64", "darwin-amd64", "darwin-arm64", "windows-amd64.exe", "windows-arm64.exe"];
for (const n of names) {
  const from = path.join(src, `nobluff-${n}`);
  if (!fs.existsSync(from)) { console.error(`missing ${from}: build the detector first`); process.exit(1); }
  fs.copyFileSync(from, path.join(dst, `nobluff-${n}`));
}
for (const f of ["LICENSE"]) fs.copyFileSync(path.join(__dirname, "..", "..", f), path.join(__dirname, "..", f));
fs.copyFileSync(path.join(src, "build-manifest.json"), path.join(__dirname, "..", "build-manifest.json"));
fs.mkdirSync(path.join(__dirname, "..", "examples"), { recursive: true });
fs.copyFileSync(path.join(src, "..", "examples", "synthetic-session.jsonl"), path.join(__dirname, "..", "examples", "synthetic-session.jsonl"));
console.log(`copied ${names.length} binaries`);
