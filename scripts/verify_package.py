"""Inspect and smoke-test the exact local tarball. Never publish or download dependencies."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def main():
    npm = shutil.which("npm.cmd" if os.name == "nt" else "npm")
    if not npm:
        raise SystemExit("npm is required")
    stage = ROOT / ".build"
    stage.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(dir=stage) as temp:
        temp = Path(temp)
        subprocess.run(["node", "scripts/copy-binaries.js"], cwd=ROOT / "npm", check=True)
        result = subprocess.check_output([npm, "pack", "--ignore-scripts", "--json", "--pack-destination", str(temp)], cwd=ROOT / "npm", text=True)
        metadata = json.loads(result)[0]
        archive = temp / metadata["filename"]
        target = temp / "unpacked"
        target.mkdir()
        with tarfile.open(archive, "r:gz") as tar:
            names = tar.getnames()
            expected = {"package/package.json", "package/README.md", "package/LICENSE", "package/build-manifest.json", "package/examples/synthetic-session.jsonl", "package/bin/nobluff.js"}
            expected |= {"package/bin/" + name for name in json.loads((ROOT / "bin/build-manifest.json").read_text())["binaries"]}
            if set(names) != expected:
                raise SystemExit(f"tarball paths differ: {set(names) ^ expected}")
            for item in tar.getmembers():
                if not item.isfile():
                    raise SystemExit("only regular files are allowed")
                dest = target / item.name
                dest.parent.mkdir(parents=True, exist_ok=True)
                dest.write_bytes(tar.extractfile(item).read())
        package = target / "package"
        manifest = json.loads((package / "build-manifest.json").read_text())
        data = json.loads((package / "package.json").read_text())
        if data["version"] != manifest["version"] or any(k in data.get("scripts", {}) for k in ("install", "postinstall", "preinstall")):
            raise SystemExit("version/lifecycle mismatch")
        for name, info in manifest["binaries"].items():
            if hashlib.sha256((package / "bin" / name).read_bytes()).hexdigest() != info["sha256"]:
                raise SystemExit(f"binary digest mismatch: {name}")
        cli = package / "bin/nobluff.js"
        version = subprocess.check_output(["node", str(cli), "version"], text=True).strip()
        if version != "nobluff " + data["version"]:
            raise SystemExit("binary version differs from package")
        check = subprocess.run(["node", str(cli), "check", str(package / "examples/synthetic-session.jsonl")], text=True, capture_output=True)
        if check.returncode != 1 or not any(f["rule"] == "tests-not-run" for f in json.loads(check.stdout)):
            raise SystemExit("packed synthetic demo did not flag the expected claim")
        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        candidate = stage / "candidate"
        candidate.mkdir(exist_ok=True)
        saved = candidate / archive.name
        saved.write_bytes(archive.read_bytes())
        result = {"package": data["name"], "version": data["version"], "sha256": digest, "file_count": len(names), "host_smoke": "passed", "lifecycle_network": "none", "candidate": str(saved.relative_to(ROOT)).replace("\\", "/")}
        (candidate / "verification.json").write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
        print(json.dumps(result))


if __name__ == "__main__":
    main()
