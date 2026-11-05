"""Keeps the demo honest: the shipped test is green, the replay exposes the bug, the fix removes it."""
import subprocess
import sys
import unittest
from pathlib import Path

EXAMPLE = Path(__file__).resolve().parent.parent / "examples" / "double-charge"
sys.path.insert(0, str(EXAMPLE))

import replay_check  # noqa: E402


class DoubleChargeDemo(unittest.TestCase):
    def test_shipped_suite_is_green(self):
        run = subprocess.run([sys.executable, "-m", "unittest", "-q", "test_payments"],
                             cwd=EXAMPLE, capture_output=True, text=True)
        self.assertEqual(run.returncode, 0, run.stderr)

    def test_replay_exposes_double_booking(self):
        self.assertEqual(replay_check.replay(replay_check.load(fixed=False)), (2, 29800))
        self.assertEqual(replay_check.main([]), 1)

    def test_fix_books_once(self):
        self.assertEqual(replay_check.replay(replay_check.load(fixed=True)), (1, 14900))
        self.assertEqual(replay_check.replay(replay_check.load(fixed=True), deliveries=5), (1, 14900))
        self.assertEqual(replay_check.main(["--fixed"]), 0)


if __name__ == "__main__":
    unittest.main()
