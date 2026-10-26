#!/usr/bin/env python3
"""Original NEXUS gate for a host-supplied govulncheck binary analysis.

This inspects a built artifact; it never executes the artifact or installs tools.
Exit 0: no known affected symbols reported; 1: affected symbols present;
2: incomplete/unsupported scan. Module matches remain visible in every report.
The scanner executable and its vulnerability database are trusted host inputs.
"""

import argparse
import hashlib
import json
import os
import pathlib
import selectors
import subprocess
import sys
import time

MAX_SCAN_BYTES = 32 << 20


class InvalidScan(ValueError):
    pass


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise InvalidScan("duplicate scanner JSON key")
        result[key] = value
    return result


def fingerprint(path):
    h = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def parse_scan(data):
    """Read the JSON object stream, not JSONL and not scanner exit status alone."""
    if not data or len(data) > MAX_SCAN_BYTES:
        raise InvalidScan("missing or oversized scanner output")
    text = data.decode("utf-8")
    decoder = json.JSONDecoder(object_pairs_hook=unique_object)
    offset = 0
    config, sbom = None, None
    findings = {}
    messages = 0
    while offset < len(text):
        while offset < len(text) and text[offset].isspace():
            offset += 1
        if offset == len(text):
            break
        try:
            message, offset = decoder.raw_decode(text, offset)
        except ValueError as exc:
            raise InvalidScan("malformed or truncated scanner output") from exc
        messages += 1
        if messages > 100000 or not isinstance(message, dict) or len(message) != 1:
            raise InvalidScan("invalid scanner message")
        kind, value = next(iter(message.items()))
        if not isinstance(value, dict):
            raise InvalidScan("invalid scanner message body")
        if kind == "config":
            if config is not None or messages != 1:
                raise InvalidScan("duplicate or misplaced scanner configuration")
            config = value
            required = {"protocol_version": "v1.0.0", "scanner_name": "govulncheck", "scan_mode": "binary", "scan_level": "symbol"}
            if any(config.get(k) != v for k, v in required.items()) or not config.get("scanner_version") or not config.get("db"):
                raise InvalidScan("unsupported scanner configuration")
        elif config is None:
            raise InvalidScan("scanner configuration must come first")
        elif kind == "SBOM":
            if sbom is not None or not isinstance(value.get("modules"), list) or not value["modules"] or not value.get("go_version"):
                raise InvalidScan("missing or duplicate artifact inventory")
            sbom = value
        elif kind == "finding":
            trace = value.get("trace")
            if not isinstance(value.get("osv"), str) or not value["osv"].startswith("GO-") or not isinstance(trace, list) or not trace or not isinstance(trace[0], dict):
                raise InvalidScan("invalid vulnerability finding")
            frame = trace[0]
            if any(not isinstance(frame.get(k), str) or not frame[k] for k in ("module", "version")) or any(k in frame and not isinstance(frame[k], str) for k in ("package", "function")) or (frame.get("function") and not frame.get("package")) or ("fixed_version" in value and not isinstance(value["fixed_version"], str)):
                raise InvalidScan("finding lacks artifact attribution")
            level = 2 if frame.get("function") else 1 if frame.get("package") else 0
            key = (value["osv"], frame["module"], frame["version"])
            item = findings.setdefault(key, {"id": key[0], "module": key[1], "version": key[2], "level": 0, "packages": set(), "symbols": set(), "fixed_versions": set()})
            item["level"] = max(item["level"], level)
            if frame.get("package"):
                item["packages"].add(frame["package"])
            if frame.get("function"):
                item["symbols"].add(frame["package"] + "." + frame["function"])
            if value.get("fixed_version"):
                item["fixed_versions"].add(value["fixed_version"])
        elif kind not in ("osv", "progress"):
            raise InvalidScan("unsupported scanner message type")
    if config is None or sbom is None:
        raise InvalidScan("scan lacks configuration or artifact inventory")
    rows = []
    for _, item in sorted(findings.items()):
        item = dict(item)
        item["evidence"] = ("module_version", "package_present", "symbol_present")[item.pop("level")]
        for key in ("packages", "symbols", "fixed_versions"):
            item[key] = sorted(item[key])
        rows.append(item)
    affected = sorted({r["id"] for r in rows if r["evidence"] == "symbol_present"})
    return {"schema": "nexus-go-binary-audit/1", "status": "affected_symbols_present" if affected else "no_known_affected_symbols", "scanner": config, "inventory": sbom, "affected_symbol_advisories": affected, "findings": rows, "raw_scan_sha256": hashlib.sha256(data).hexdigest()}


def audit(binary, scanner, timeout):
    before = fingerprint(binary)
    # Linux/Docker host tool: bound output while reading, not after capture.
    command = [scanner, "-json", "-mode", "binary", str(binary)]
    deadline = time.monotonic() + timeout
    data = bytearray()
    with subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, env={**os.environ, "GOTELEMETRY": "off"}) as process:
        try:
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                while True:
                    remaining = deadline - time.monotonic()
                    if remaining <= 0 or not selector.select(remaining):
                        raise subprocess.TimeoutExpired(command, timeout)
                    chunk = os.read(process.stdout.fileno(), 65536)
                    if not chunk:
                        break
                    if len(data) + len(chunk) > MAX_SCAN_BYTES:
                        raise InvalidScan("scanner output limit exceeded")
                    data.extend(chunk)
            result = process.wait(timeout=max(0.001, deadline - time.monotonic()))
        finally:
            if process.poll() is None:
                process.kill()
                process.wait()
    if result != 0:
        raise InvalidScan("scanner did not complete successfully")
    report = parse_scan(bytes(data))
    if fingerprint(binary) != before:
        raise InvalidScan("artifact changed during analysis")
    report["artifact"] = {"name": binary.name, "sha256": before}
    return report


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary", type=pathlib.Path)
    parser.add_argument("--scanner", default="govulncheck")
    parser.add_argument("--timeout", type=int, default=120)
    args = parser.parse_args(argv)
    if not 1 <= args.timeout <= 600:
        parser.error("timeout must be between 1 and 600 seconds")
    try:
        report = audit(args.binary.resolve(strict=True), args.scanner, args.timeout)
    except (OSError, ValueError, UnicodeError, subprocess.TimeoutExpired):
        # Scanner/provider errors may include private paths or credentials.
        json.dump({"schema": "nexus-go-binary-audit/1", "status": "incomplete_scan"}, sys.stdout)
        sys.stdout.write("\n")
        return 2
    json.dump(report, sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 1 if report["affected_symbol_advisories"] else 0


if __name__ == "__main__":
    sys.exit(main())
