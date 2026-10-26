import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from curriculum import EVAL, TRAIN, messages, validate
from run import evaluate, export, grade, NoRedirect, select_lessons, teaching_messages


class AcademyTests(unittest.TestCase):
    def test_export_excludes_evaluation_and_has_provenance(self):
        validate()
        with tempfile.TemporaryDirectory() as directory:
            out = Path(directory)
            export(out)
            rows = [json.loads(x) for x in (out / "train.jsonl").read_text().splitlines()]
            self.assertEqual(len(rows), len(TRAIN))
            prompts = {x["messages"][1]["content"] for x in rows}
            self.assertTrue(all(messages(x)[1]["content"] not in prompts for x in EVAL))
            self.assertEqual(len(json.loads((out / "provenance.json").read_text())["examples"]), len(TRAIN))
            with self.assertRaises(FileExistsError):
                export(out)

    def test_grade_rejects_hallucinated_evidence_and_malformed_json(self):
        item = EVAL[0]
        expected = item["answer"]
        self.assertTrue(grade(expected, expected, item["evidence"])["pass"])
        for ids in [["E9"], ["E1", "E1"], ["E1", "E2"], [], [None], {}]:
            self.assertFalse(grade(dict(expected, evidence_ids=ids), expected, item["evidence"])["pass"])
        for answer in [[], None, {}, dict(expected, finding=[]), dict(expected, action={}), dict(expected, explanation="")]:
            self.assertFalse(grade(answer, expected, item["evidence"])["pass"])

    def test_model_never_receives_answer_or_previous_evaluation(self):
        calls = []
        def fake_request(path, payload=None, timeout=120):
            if path == "/api/tags":
                return {"models": [{"name": "test"}]}
            if path in {"/api/version", "/api/show"}:
                return {}
            calls.append(payload)
            return {"done": True, "message": {"content": json.dumps(EVAL[len(calls)-1]["answer"])}}
        with tempfile.TemporaryDirectory() as directory, patch("run.request", side_effect=fake_request):
            evaluate(Path(directory), "test", 2, 1)
        self.assertEqual(len(calls), 2)
        for i, call in enumerate(calls):
            self.assertEqual(call["messages"], messages(EVAL[i]))
            self.assertNotIn("answer", json.loads(call["messages"][1]["content"]))
            self.assertNotIn("tools", call)

    def test_transport_failure_counts_as_failure(self):
        def fake_request(path, payload=None, timeout=120):
            if path == "/api/tags":
                return {"models": [{"name": "test"}]}
            if path == "/api/chat":
                raise TimeoutError("test timeout")
            return {}
        with tempfile.TemporaryDirectory() as directory, patch("run.request", side_effect=fake_request):
            out = Path(directory)
            evaluate(out, "test", 1, 1)
            summary = json.loads((out / "summary.json").read_text())
            self.assertEqual((summary["total"], summary["passed"], summary["errors"]), (1, 0, 1))

    def test_redirects_rejected(self):
        with self.assertRaises(ValueError):
            NoRedirect().redirect_request(None)

    def test_extended_export_never_contains_evaluation(self):
        import extended_curriculum as extended
        extended.validate()
        with tempfile.TemporaryDirectory() as directory:
            out = Path(directory)
            export(out, "extended-v2")
            rows = [json.loads(x) for x in (out / "train.jsonl").read_text().splitlines()]
            self.assertEqual(len(rows), len(extended.TRAIN))
            prompts = {row["messages"][1]["content"] for row in rows}
            self.assertTrue(all(messages(x)[1]["content"] not in prompts for x in extended.EVAL + EVAL))

    def test_retrieval_cannot_read_evaluation_answers(self):
        import copy
        import extended_curriculum as extended
        for item in extended.EVAL:
            selected = select_lessons(item, extended.TRAIN)
            changed = copy.deepcopy(item)
            changed["answer"] = {"injected": "never use this"}
            self.assertEqual(selected, select_lessons(changed, extended.TRAIN))
            self.assertTrue(0 < len(selected) <= 3)
            actual, ids = teaching_messages(item, extended.TRAIN, "system", "lessons-v1")
            self.assertEqual(ids, [x["id"] for x in selected])
            self.assertEqual(actual[-1]["content"], messages(item)[1]["content"])
            self.assertTrue(all(x in extended.TRAIN for x in selected))
            self.assertEqual(select_lessons(item, extended.TRAIN, char_budget=0), [])

    def test_model_system_is_preserved(self):
        calls = []
        def fake(path, payload=None, timeout=120):
            if path == "/api/tags":
                return {"models": [{"name": "test"}]}
            if path == "/api/show":
                return {"system": "Scope owned by model host"}
            if path == "/api/version":
                return {}
            calls.append(payload)
            return {"done": True, "message": {"content": json.dumps(EVAL[0]["answer"])}}
        with tempfile.TemporaryDirectory() as directory, patch("run.request", side_effect=fake):
            evaluate(Path(directory), "test", 1, 1)
        self.assertTrue(calls[0]["messages"][0]["content"].startswith("Scope owned by model host"))


if __name__ == "__main__":
    unittest.main()
