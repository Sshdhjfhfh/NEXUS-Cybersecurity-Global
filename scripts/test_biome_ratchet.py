import json
import tempfile
import unittest
from pathlib import Path

import biome_ratchet as br


def report(*diags):
    return {"diagnostics": [{"severity": sev, "location": {"path": {"file": f}}} for f, sev in diags]}


class BiomeRatchetTest(unittest.TestCase):
    def test_counts_only_errors(self):
        r = report(("a.tsx", "error"), ("a.tsx", "error"), ("a.tsx", "warning"), ("b.ts", "information"))
        self.assertEqual(br.error_counts(r), {"a.tsx": 2})

    def test_regression_and_new_file_fail(self):
        worse, better = br.compare({"a.tsx": 3, "new.tsx": 1, "c.tsx": 0}, {"a.tsx": 2, "c.tsx": 4})
        self.assertEqual(worse, [("a.tsx", 3, 2), ("new.tsx", 1, 0)])
        self.assertEqual(better, [("c.tsx", 0, 4)])

    def test_update_never_raises_existing_file(self):
        new = br.updated_baseline({"a.tsx": 5, "b.tsx": 1, "n.tsx": 2}, {"a.tsx": 3, "b.tsx": 4, "gone.tsx": 9})
        self.assertEqual(new, {"a.tsx": 3, "b.tsx": 1, "n.tsx": 2})

    def test_main_exit_codes(self):
        with tempfile.TemporaryDirectory() as d:
            base = Path(d, "base.json")
            base.write_text(json.dumps({"a.tsx": 1}))
            rep = Path(d, "r.json")
            rep.write_text(json.dumps(report(("a.tsx", "error"))))
            self.assertEqual(br.main(["--report", str(rep), "--baseline", str(base)]), 0)
            rep.write_text(json.dumps(report(("a.tsx", "error"), ("a.tsx", "error"))))
            self.assertEqual(br.main(["--report", str(rep), "--baseline", str(base)]), 1)


if __name__ == "__main__":
    unittest.main()
