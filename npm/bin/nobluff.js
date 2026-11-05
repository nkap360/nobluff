#!/usr/bin/env node
// Runs the NoBluff detector binary that matches this machine. No network, no install script.
"use strict";
const { spawnSync } = require("child_process");
const path = require("path");
const fs = require("fs");

const os = { win32: "windows", darwin: "darwin", linux: "linux" }[process.platform];
const arch = { x64: "amd64", arm64: "arm64" }[process.arch];
if (!os || !arch) {
  console.error(`nobluff: no binary for ${process.platform}/${process.arch}. Build it from https://github.com/provenwork/nobluff/tree/main/detector`);
  process.exit(2);
}
const bin = path.join(__dirname, `nobluff-${os}-${arch}${os === "windows" ? ".exe" : ""}`);
if (!fs.existsSync(bin)) {
  console.error(`nobluff: missing ${path.basename(bin)} in this package.`);
  process.exit(2);
}
try { fs.chmodSync(bin, 0o755); } catch (_) { /* read-only install: fine if already executable */ }
const r = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });
if (r.error) {
  console.error(`nobluff: ${r.error.message}`);
  process.exit(2);
}
process.exit(r.status === null ? 2 : r.status);
