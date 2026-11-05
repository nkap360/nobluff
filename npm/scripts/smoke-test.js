// `npm test`: the launcher runs the right binary and the detector blocks a bluff.
"use strict";
const { spawnSync } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");
const cli = path.join(__dirname, "..", "bin", "nobluff.js");
const v = spawnSync(process.execPath, [cli, "version"], { encoding: "utf8" });
if (v.status !== 0 || !/nobluff \d+\.\d+\.\d+/.test(v.stdout)) { console.error("version failed", v.stdout, v.stderr); process.exit(1); }
const t = path.join(fs.mkdtempSync(path.join(os.tmpdir(), "nobluff-")), "t.jsonl");
fs.writeFileSync(t, [
  { type: "assistant", message: { role: "assistant", content: [{ type: "tool_use", id: "e1", name: "Edit", input: { file_path: "a.py" } }] } },
  { type: "user", message: { role: "user", content: [{ type: "tool_result", tool_use_id: "e1", content: "ok" }] } },
  { type: "assistant", message: { role: "assistant", content: [{ type: "text", text: "All tests pass." }] } },
].map((e) => JSON.stringify(e)).join("\n"));
const c = spawnSync(process.execPath, [cli, "check", t], { encoding: "utf8" });
if (c.status !== 1 || !c.stdout.includes("tests-not-run")) { console.error("check failed", c.status, c.stdout, c.stderr); process.exit(1); }
console.log(`ok: ${v.stdout.trim()}, bluff detected`);
