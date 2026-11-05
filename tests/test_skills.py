"""Structural checks on the pack: every skill loads, the plugin manifest is valid."""
import json
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EXPECTED = {"nobluff", "evidence-first", "no-placeholders", "fact-gate", "verdict-format", "self-audit"}


def frontmatter(path: Path) -> dict:
    """Read the YAML frontmatter, rejecting what a real YAML parser rejects.

    An unquoted value containing ": " is not valid YAML; Claude Code then drops the skill from the
    model-invocable list without any error. That bug shipped in 0.1.0 and 0.2.0 because this reader
    used to split on the first colon and accept it.
    """
    text = path.read_text(encoding="utf-8")
    m = re.match(r"^---\n(.*?)\n---\n", text.replace("\r\n", "\n"), re.S)
    if not m:
        return {}
    out = {}
    for line in m.group(1).splitlines():
        if ":" in line and not line.startswith(" "):
            key, value = line.split(":", 1)
            value = value.strip()
            quoted = len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'"
            if not quoted and (": " in value or value.endswith(":") or " #" in value):
                raise ValueError(f"{path}: unquoted '{key}' value is invalid YAML; wrap it in double quotes")
            out[key.strip()] = value[1:-1] if quoted else value
    return out


class SkillPack(unittest.TestCase):
    def test_every_expected_skill_exists(self):
        found = {p.parent.name for p in (ROOT / "skills").glob("*/SKILL.md")}
        self.assertEqual(found, EXPECTED)

    def test_skills_have_name_and_description(self):
        for path in (ROOT / "skills").glob("*/SKILL.md"):
            fm = frontmatter(path)
            with self.subTest(skill=path.parent.name):
                self.assertEqual(fm.get("name"), path.parent.name)
                self.assertGreater(len(fm.get("description", "")), 20)

    def test_root_skill_has_frontmatter(self):
        fm = frontmatter(ROOT / "SKILL.md")
        self.assertTrue(fm.get("name"))
        self.assertTrue(fm.get("description"))

    def test_plugin_manifest(self):
        manifest = json.loads((ROOT / ".claude-plugin" / "plugin.json").read_text(encoding="utf-8"))
        self.assertEqual(manifest["name"], "nobluff")
        self.assertRegex(manifest["version"], r"^\d+\.\d+\.\d+$")
        self.assertEqual(manifest["license"], "MIT")

    def test_stop_hook_points_to_the_launcher(self):
        hooks = json.loads((ROOT / "hooks" / "hooks.json").read_text(encoding="utf-8"))
        command = hooks["hooks"]["Stop"][0]["hooks"][0]["command"]
        self.assertIn("${CLAUDE_PLUGIN_ROOT}/bin/nobluff", command)
        self.assertTrue((ROOT / "bin" / "nobluff").exists())

    def test_a_binary_ships_for_every_supported_platform(self):
        for name in ["linux-amd64", "linux-arm64", "darwin-amd64", "darwin-arm64", "windows-amd64.exe", "windows-arm64.exe"]:
            with self.subTest(platform=name):
                binary = ROOT / "bin" / f"nobluff-{name}"
                self.assertTrue(binary.exists(), binary)
                self.assertGreater(binary.stat().st_size, 500_000)

    def test_changelog_mentions_current_version(self):
        manifest = json.loads((ROOT / ".claude-plugin" / "plugin.json").read_text(encoding="utf-8"))
        self.assertIn(f"## {manifest['version']}", (ROOT / "CHANGELOG.md").read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
