#!/usr/bin/env python3
"""Run the actual compiled NEXUS application against a fresh local lab database.

No LLM credentials/configuration are inherited. This is a release smoke test,
not proof of offensive effectiveness, tenant isolation or general AI ability.
"""

import argparse
from contextlib import closing
import hashlib
import http.client
import json
import math
import os
import pathlib
import re
import secrets
import socket
import statistics
import subprocess
import tempfile
import time
from urllib.parse import parse_qs, urlsplit


def lab_dsn(value):
    p = urlsplit(value)
    if p.scheme not in ("postgres", "postgresql") or not re.fullmatch(r"/nexus_lab_[a-zA-Z0-9_]{1,40}", p.path):
        raise ValueError("a dedicated nexus_lab_* database is required")
    query = parse_qs(p.query, strict_parsing=True)
    if set(query) - {"host", "sslmode"} or any(len(v) != 1 for v in query.values()):
        raise ValueError("unsupported laboratory database parameters")
    if p.hostname in ("127.0.0.1", "::1") and "host" not in query:
        return value
    if p.hostname is None and "host" in query:
        host = pathlib.Path(query["host"][0]).resolve(strict=True)
        if host.is_dir() and host.is_relative_to("/tmp") and host.name.startswith("nexus-"):
            return value
    raise ValueError("only a local dedicated laboratory database is allowed")


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def request(port, method, path, body=None, token=None):
    headers = {}
    if body is not None:
        body = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    if token is not None:
        headers["Authorization"] = "Bearer " + token
    with closing(http.client.HTTPConnection("127.0.0.1", port, timeout=5)) as conn:
        start = time.perf_counter()
        conn.request(method, path, body, headers)
        response = conn.getresponse()
        data = response.read(8 << 20)
        elapsed = (time.perf_counter() - start) * 1000
        return response.status, data, elapsed


def stop(process):
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)
    if process.returncode != 0:
        raise RuntimeError("NEXUS did not shut down cleanly")


def wait_ready(process, port):
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError("NEXUS exited before readiness")
        try:
            status, body, _ = request(port, "GET", "/api/health")
            if status == 200 and json.loads(body).get("ok"):
                return
        except (OSError, ValueError):
            pass
        time.sleep(0.1)
    raise RuntimeError("NEXUS readiness deadline exceeded")


def run(binary, dsn, report):
    binary = binary.resolve(strict=True)
    report["artifact_sha256"] = hashlib.sha256(binary.read_bytes()).hexdigest()
    report["kind"] = "real-process-local-postgresql-http"
    report["checks"] = []
    with tempfile.TemporaryDirectory(prefix="nexus-release-") as directory:
        root = pathlib.Path(directory)
        api, proxy = free_port(), free_port()
        while proxy == api:
            proxy = free_port()
        # Explicit environment: no inherited API keys, providers or production config.
        env = {"PATH": os.defpath, "LANG": "C.UTF-8", "NEXUS_PG_DSN": lab_dsn(dsn), "NEXUS_CONFIG": str(root / "absent-config.json"), "NEXUS_SKILL_DIR": str(root / "skills")}
        command = [str(binary), "-addr", f"127.0.0.1:{api}", "-proxy", f"127.0.0.1:{proxy}", "-data", str(root / "data")]
        with (root / "server.log").open("wb") as log:
            def start():
                child = subprocess.Popen(command, cwd=root, env=env, stdout=log, stderr=subprocess.STDOUT)
                try:
                    wait_ready(child, api)
                except Exception:
                    if child.poll() is None:
                        child.terminate()
                        try:
                            child.wait(timeout=10)
                        except subprocess.TimeoutExpired:
                            child.kill()
                            child.wait()
                    raise
                return child

            def check(name, status, expected):
                report["checks"].append({"name": name, "status": status, "expected": expected, "passed": status == expected})
                if status != expected:
                    raise RuntimeError("failed laboratory check: " + name)

            child = start()
            try:
                check("unauthenticated_tasks", request(api, "GET", "/api/tasks")[0], 401)
                check("invalid_token", request(api, "GET", "/api/tasks", token="invalid-lab-token")[0], 401)
                status, html, _ = request(api, "GET", "/login/")
                check("embedded_login_page", status, 200)
                if b"<html" not in html.lower():
                    raise RuntimeError("embedded frontend is missing")
                assets = sorted({s.decode() for s in re.findall(rb'(?:src|href)="(/_next/static/[^"?]+)', html)})
                if len(assets) < 2:
                    raise RuntimeError("compiled frontend assets are missing")
                for index, asset in enumerate(assets[:3]):
                    status, content, _ = request(api, "GET", asset)
                    check("embedded_asset_" + str(index), status, 200)
                    if not content:
                        raise RuntimeError("empty compiled asset")
                status, data, _ = request(api, "GET", "/api/auth/status")
                check("fresh_database_auth_status", status, 200)
                if json.loads(data).get("initialized") is not False:
                    raise RuntimeError("database is not fresh; refusing to initialize credentials")
                password = secrets.token_urlsafe(24)
                status, data, _ = request(api, "POST", "/api/auth/init", {"password": password})
                check("initialize_lab_account", status, 200)
                token = json.loads(data)["token"]
                check("prevent_account_reinitialization", request(api, "POST", "/api/auth/init", {"password": secrets.token_urlsafe(24)})[0], 403)
                check("wrong_password", request(api, "POST", "/api/auth/login", {"username": "ARTEX", "password": "incorrect-lab-password"})[0], 401)
                check("valid_password", request(api, "POST", "/api/auth/login", {"username": "ARTEX", "password": password})[0], 200)
                check("authenticated_tasks", request(api, "GET", "/api/tasks", token=token)[0], 200)
                status, data, _ = request(api, "GET", "/api/lab/status", token=token)
                check("laboratory_status", status, 200)
                readiness = json.loads(data)
                if readiness.get("ready") is not False or "authorization" not in readiness.get("blockers", []):
                    raise RuntimeError("laboratory controls were unexpectedly bypassed")
                report["laboratory_blockers"] = readiness["blockers"]
                status, data, _ = request(api, "GET", "/api/stats", token=token)
                check("runtime_stats", status, 200)
                if json.loads(data).get("llm_configured") is not False:
                    raise RuntimeError("unexpected LLM configuration in isolated lab")
                latencies = []
                for _ in range(20):
                    status, _, elapsed = request(api, "GET", "/api/health")
                    if status != 200:
                        raise RuntimeError("health request failed during measurement")
                    latencies.append(elapsed)
                report["health_latency_ms"] = {"samples": len(latencies), "p50": statistics.median(latencies), "p95": sorted(latencies)[math.ceil(len(latencies) * 0.95) - 1]}
                memory = pathlib.Path(f"/proc/{child.pid}/status").read_text()
                rss = re.search(r"^VmRSS:\s+(\d+)\s+kB", memory, re.M)
                report["rss_sample_kib"] = int(rss[1]) if rss else None
            finally:
                stop(child)
            child = start()
            try:
                status, data, _ = request(api, "GET", "/api/auth/status")
                check("persisted_account_after_restart", status, 200)
                if json.loads(data).get("initialized") is not True:
                    raise RuntimeError("account initialization did not persist")
                check("existing_session_after_restart", request(api, "GET", "/api/tasks", token=token)[0], 200)
                check("password_after_restart", request(api, "POST", "/api/auth/login", {"username": "ARTEX", "password": password})[0], 200)
            finally:
                stop(child)
        report["server_log_sha256"] = hashlib.sha256((root / "server.log").read_bytes()).hexdigest()
    report["passed"] = True
    report["scope"] = "startup, static assets, authentication, persistence, restart and fail-closed lab readiness; not full production certification"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=pathlib.Path, required=True)
    parser.add_argument("--report", type=pathlib.Path, required=True)
    args = parser.parse_args()
    report = {"schema": "nexus-release-smoke/1", "passed": False}
    try:
        run(args.binary, os.environ.get("NEXUS_LAB_PG_DSN", ""), report)
    except (OSError, ValueError, KeyError, RuntimeError, subprocess.SubprocessError) as exc:
        report["error"] = str(exc) if isinstance(exc, RuntimeError) else type(exc).__name__
    args.report.write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"passed": report["passed"], "checks": len(report.get("checks", []))}))
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
