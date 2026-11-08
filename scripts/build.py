"""Reproduce or compare all six shipped binaries with the pinned local compiler. No network."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
TOOLCHAIN = "go1.27.3"
TARGETS = [(system, arch) for system in ("linux", "darwin", "windows") for arch in ("amd64", "arm64")]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="compare rebuilt bytes; do not replace shipped files")
    args = parser.parse_args()
    env = dict(os.environ, CGO_ENABLED="0", GOTOOLCHAIN="local", GOWORK="off", GOFLAGS="", GOEXPERIMENT="", GOPROXY="off", GOSUMDB="off", GOTELEMETRY="off", GOAMD64="v1", GOARM64="v8.0")
    version = subprocess.check_output(["go", "version"], text=True, env=env).split()[2]
    if version != TOOLCHAIN:
        raise SystemExit(f"need {TOOLCHAIN}, found {version}; install compiler explicitly")
    (ROOT / ".build").mkdir(exist_ok=True)
    manifest = {"version": json.loads((ROOT / "npm/package.json").read_text())["version"], "toolchain": TOOLCHAIN,
        "flags": ["-buildvcs=false", "-trimpath", "-ldflags=-s -w"],
        "sources": {str(p.relative_to(ROOT)).replace("\\", "/"): digest(p) for p in sorted((ROOT / "detector").glob("*")) if p.name == "go.mod" or p.suffix == ".go" and not p.name.endswith("_test.go")}, "binaries": {}}
    with tempfile.TemporaryDirectory(dir=ROOT / ".build") as temp:
        for system, arch in TARGETS:
            name = f"nobluff-{system}-{arch}" + (".exe" if system == "windows" else "")
            target = Path(temp) / name
            subprocess.run(["go", "build", "-buildvcs=false", "-trimpath", "-ldflags=-s -w", "-o", str(target), "."], cwd=ROOT / "detector", env=dict(env, GOOS=system, GOARCH=arch), check=True)
            shipped = ROOT / "bin" / name
            if args.check:
                if not shipped.exists() or target.read_bytes() != shipped.read_bytes():
                    raise SystemExit(f"source/binary mismatch: {name}")
            else:
                shipped.write_bytes(target.read_bytes())
                shipped.chmod(0o755)
            manifest["binaries"][name] = {"sha256": digest(target), "bytes": target.stat().st_size}
    body = json.dumps(manifest, indent=2, sort_keys=True) + "\n"
    path = ROOT / "bin/build-manifest.json"
    if args.check:
        if not path.exists() or path.read_text(encoding="utf-8") != body:
            raise SystemExit("build manifest mismatch")
    else:
        path.write_text(body, encoding="utf-8")
    print(f"{len(TARGETS)} binaries {'compared byte for byte' if args.check else 'built'}; {TOOLCHAIN}")


if __name__ == "__main__":
    main()
