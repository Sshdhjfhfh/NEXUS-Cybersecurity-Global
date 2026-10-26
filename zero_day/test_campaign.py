import base64
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import campaign


def guidance(**updates):
    obj = {"source": "test-model", "rationale": "Test parser boundaries",
           "seeds_base64": [base64.b64encode(b"HELLO").decode()],
           "dictionary": ["NEXUS", "!"]}
    obj.update(updates)
    return json.dumps(obj).encode()


class GuidanceTests(unittest.TestCase):
    def test_valid_guidance_deduplicates_seeds(self):
        obj, seeds = campaign.parse_guidance(guidance(seeds_base64=["SEVMTE8=", "SEVMTE8="]))
        self.assertEqual(seeds, [b"HELLO"])
        self.assertEqual(obj["source"], "test-model")

    def test_rejects_unknown_fields_malformed_and_large_inputs(self):
        invalid = [b"[]", b"{", guidance(command="echo bypass"), guidance(source=""),
                   guidance(seeds_base64=[]), guidance(seeds_base64=["invalid*"]),
                   guidance(seeds_base64=[""]), guidance(seeds_base64=[1]),
                   guidance(seeds_base64=[base64.b64encode(b"x" * 4097).decode()]),
                   guidance(dictionary=["x" * 65]), guidance(dictionary=[1]),
                   guidance(dictionary=["a"] * 65), guidance(seeds_base64=["YQ=="] * 65),
                   b" " * (campaign.MAX_GUIDANCE + 1),
                   b'{"source":"first","source":"second"}']
        for data in invalid:
            with self.subTest(data=data[:80]), self.assertRaises(ValueError):
                campaign.parse_guidance(data)

    def test_dictionary_escapes_shell_and_afl_metacharacters(self):
        text = campaign.afl_dictionary(['"\\\n$(touch /tmp/no)', "é", "é"])
        self.assertNotIn("$(", text)
        self.assertNotIn("touch", text)
        self.assertIn(r"\x22\x5c\x0a", text)
        self.assertIn(r"\xc3\xa9", text)
        self.assertEqual(len(text.splitlines()), 2)


class CampaignTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.target = self.root / "target"
        self.target.write_bytes(b"fake target for orchestration fixture")

    def test_failed_afl_saves_a_report_and_never_uses_a_shell(self):
        class FailedProcess:
            def __init__(self, args, **kwargs):
                self.args, self.kwargs = args, kwargs
                self.returncode = 1
                assert not kwargs.get("shell")
                assert args[-1] == "@@"

            def wait(self, **kwargs):
                return 1

            def poll(self):
                return self.returncode

        with patch.object(campaign, "replay", return_value={"outcome": "ok"}), \
                patch.object(campaign.subprocess, "Popen", FailedProcess):
            report = campaign.campaign(1, work=self.root / "runs", target=self.target)
        self.assertEqual(report["status"], "failed")
        session = self.root / "runs" / report["session_id"]
        self.assertTrue((session / "report.html").is_file())
        self.assertIn("AFL++", json.loads((session / "report.json").read_text())["error"])

    def test_crashing_guidance_is_not_claimed_as_fuzzer_discovery(self):
        with patch.object(campaign, "replay", return_value={"outcome": "crash", "sha256": "sample", "bytes": 5}), \
                patch.object(campaign.subprocess, "Popen") as launch:
            report = campaign.campaign(1, work=self.root / "runs", target=self.target)
        launch.assert_not_called()
        self.assertEqual(report["status"], "failed")
        self.assertEqual(report["reproduced_crashes"], 0)
        self.assertEqual(len(report["rejected_seeds"]), 1)

    def test_hangs_are_distinct_from_crashes_and_nonzero_exits(self):
        sample = self.root / "sample"
        sample.write_bytes(b"HELLO")
        for code, outcome in [(0, "ok"), (2, "error"), (-6, "crash")]:
            with self.subTest(code=code), patch.object(campaign.subprocess, "run", return_value=subprocess.CompletedProcess([], code)):
                self.assertEqual(campaign.replay(sample)["outcome"], outcome)
        with patch.object(campaign.subprocess, "run", side_effect=subprocess.TimeoutExpired("target", 2)):
            self.assertEqual(campaign.replay(sample)["outcome"], "timeout")

    def test_runtime_limits_and_session_traversal(self):
        for seconds in [-1, 0, 601]:
            with self.subTest(seconds=seconds), self.assertRaises(ValueError):
                campaign.campaign(seconds, work=self.root)
        for session in ["../other", "/tmp", "a" * 31, "x" * 32]:
            with self.subTest(session=session), self.assertRaises(ValueError):
                campaign.load_session(self.root, session)

    def test_html_escapes_model_generated_text(self):
        report = {"status": "failed", "reproduced_crashes": 0, "findings": [],
                  "guidance": {"rationale": "<script>alert(1)</script>"}}
        campaign.save_report(self.root, report)
        page = (self.root / "report.html").read_text()
        self.assertNotIn("<script>", page)
        self.assertIn("&lt;script&gt;", page)


if __name__ == "__main__":
    unittest.main()
