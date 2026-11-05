"""Structural checks on the pack: every skill loads, the plugin manifest is valid."""
import json
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EXPECTED = {"nobluff", "evidence-first", "no-placeholders", "fact-gate", "verdict-format", "self-audit"}


def frontmatter(path: Path) -> dict:
    text = path.read_text(encoding="utf-8")
    m = re.match(r"^---\n(.*?)\n---\n", text.replace("\r\n", "\n"), re.S)
    if not m:
        return {}
    out = {}
    for line in m.group(1).splitlines():
        if ":" in line and not line.startswith(" "):
            key, value = line.split(":", 1)
            out[key.strip()] = value.strip().strip('"')
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

    def test_changelog_mentions_current_version(self):
        manifest = json.loads((ROOT / ".claude-plugin" / "plugin.json").read_text(encoding="utf-8"))
        self.assertIn(f"## {manifest['version']}", (ROOT / "CHANGELOG.md").read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
