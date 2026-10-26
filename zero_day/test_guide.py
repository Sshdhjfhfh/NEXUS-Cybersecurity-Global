import io
import json
import unittest
from unittest.mock import patch

import guide


class LocalGuidanceTests(unittest.TestCase):
    def test_rejects_external_hosts_credentials_and_paths(self):
        for endpoint in ["https://127.0.0.1", "http://example.com", "http://127.0.0.1.example.com",
                         "http://user:pass@localhost:11434", "http://localhost/api",
                         "http://localhost?redirect=cloud"]:
            with self.subTest(endpoint=endpoint), self.assertRaises(ValueError):
                guide.generate("local-model", "parser source", endpoint=endpoint)

    def test_local_response_is_validated_and_records_actual_model(self):
        response = {"source": "untrusted-name", "rationale": "Parser boundary",
                    "seeds_base64": ["SEVMTE8="], "dictionary": ["!"]}
        payload = json.dumps({"response": json.dumps(response)}).encode()
        with patch.object(guide, "build_opener") as opener:
            opener.return_value.open.return_value = io.BytesIO(payload)
            obj = guide.generate("local-model", "parser source", "execs_done: 10")
            request = opener.return_value.open.call_args.args[0]
            self.assertEqual(request.full_url, "http://127.0.0.1:11434/api/generate")
            self.assertFalse(json.loads(request.data)["stream"])
            self.assertEqual(obj["source"], "local-model")

    def test_model_cannot_inject_execution_options(self):
        response = {"source": "model", "rationale": "test", "seeds_base64": ["SEVMTE8="],
                    "dictionary": [], "command": "shell command"}
        with patch.object(guide, "build_opener") as opener:
            opener.return_value.open.return_value = io.BytesIO(json.dumps({"response": json.dumps(response)}).encode())
            with self.assertRaises(ValueError):
                guide.generate("local-model", "parser source")


if __name__ == "__main__":
    unittest.main()
