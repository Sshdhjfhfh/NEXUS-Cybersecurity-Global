import json
import unittest

from audit_go_binary import InvalidScan, parse_scan


CONFIG = {"config": {"protocol_version": "v1.0.0", "scanner_name": "govulncheck", "scanner_version": "v1.9.0", "db": "https://vuln.go.dev", "scan_mode": "binary", "scan_level": "symbol"}}
SBOM = {"SBOM": {"go_version": "go1.26.10", "modules": [{"path": "example.com/fixture", "version": "v1.0.0"}]}}


def stream(*messages):
    return "\n".join(json.dumps(m, indent=2) for m in messages).encode()


def finding(package=None, function=None):
    frame = {"module": "stdlib", "version": "v1.26.3"}
    if package:
        frame["package"] = package
    if function:
        frame["function"] = function
    return {"finding": {"osv": "GO-2026-6809", "trace": [frame], "fixed_version": "v1.26.10"}}


class BinaryAuditTests(unittest.TestCase):
    def test_symbol_findings_fail_gate_even_in_successful_json_stream(self):
        result = parse_scan(stream(CONFIG, SBOM, finding(), finding("crypto/x509"), finding("crypto/x509", "Verify"), finding("crypto/x509", "Verify")))
        self.assertEqual(result["status"], "affected_symbols_present")
        self.assertEqual(result["affected_symbol_advisories"], ["GO-2026-6809"])
        self.assertEqual(len(result["findings"]), 1)
        self.assertEqual(result["findings"][0]["symbols"], ["crypto/x509.Verify"])

    def test_module_match_stays_visible_without_claiming_symbol_reachability(self):
        result = parse_scan(stream(CONFIG, SBOM, finding()))
        self.assertEqual(result["status"], "no_known_affected_symbols")
        self.assertEqual(result["findings"][0]["evidence"], "module_version")

    def test_empty_findings_require_artifact_inventory(self):
        self.assertEqual(parse_scan(stream(CONFIG, SBOM))["status"], "no_known_affected_symbols")
        for data in [b"", stream(CONFIG), stream(SBOM), b"null", stream(CONFIG, SBOM) + b"{"]:
            with self.subTest(data=data), self.assertRaises(InvalidScan):
                parse_scan(data)

    def test_unsupported_modes_and_duplicate_configuration_rejected(self):
        for key, value in [("protocol_version", "future"), ("scan_mode", "source"), ("scan_level", "module")]:
            config = {"config": {**CONFIG["config"], key: value}}
            with self.subTest(key=key), self.assertRaises(InvalidScan):
                parse_scan(stream(config, SBOM))
        for data in [stream(CONFIG, CONFIG, SBOM), stream(CONFIG, SBOM, SBOM), stream(CONFIG, {"unknown": {}}, SBOM)]:
            with self.assertRaises(InvalidScan):
                parse_scan(data)

    def test_malformed_finding_rejected_instead_of_silently_ignored(self):
        for value in [{}, {"osv": "GO-2026-6809", "trace": []}, {"osv": "GO-2026-6809", "trace": [{"module": "stdlib", "version": "v1.26.3", "function": "Verify"}]}]:
            with self.subTest(value=value), self.assertRaises(InvalidScan):
                parse_scan(stream(CONFIG, SBOM, {"finding": value}))

    def test_duplicate_keys_cannot_hide_a_finding(self):
        duplicate = b'{"finding":{},"finding":{}}'
        with self.assertRaises(InvalidScan):
            parse_scan(stream(CONFIG, SBOM) + duplicate)


if __name__ == "__main__":
    unittest.main()
