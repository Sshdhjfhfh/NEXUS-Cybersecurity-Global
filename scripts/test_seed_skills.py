import tempfile
import unittest
from pathlib import Path

from seed_skills import seed


class SeedTests(unittest.TestCase):
    def test_preserves_user_packages_and_seeds_new_ones(self):
        with tempfile.TemporaryDirectory() as tmp:
            source, destination = Path(tmp) / "bundled", Path(tmp) / "user"
            (source / "browser").mkdir(parents=True)
            (source / "browser" / "SKILL.md").write_text("version one")
            self.assertEqual(seed(source, destination), 1)
            (destination / "browser" / "SKILL.md").write_text("user version")
            (source / "browser" / "SKILL.md").write_text("version two")
            self.assertEqual(seed(source, destination), 0)
            self.assertEqual((destination / "browser" / "SKILL.md").read_text(), "user version")

    def test_rejects_destination_symlink(self):
        with tempfile.TemporaryDirectory() as tmp:
            source, outside, destination = [Path(tmp) / name for name in ("source", "outside", "dest")]
            source.mkdir()
            outside.mkdir()
            destination.symlink_to(outside, target_is_directory=True)
            with self.assertRaises(ValueError):
                seed(source, destination)


if __name__ == "__main__":
    unittest.main()
