"""Offline AFL++ campaigns, seed guidance and crash replay in a lab container.

The target is a file parser at a fixed path; no URLs, shell commands, network
clients or production-control defaults are accepted. All artifacts are local.
"""
import argparse
import base64
import hashlib
import html
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys
import time
import uuid

MAX_INPUT = 4096
MAX_GUIDANCE = 1024 * 1024
TARGET = Path("/opt/zero-day/target")
WORK = Path("/work")
SESSION_ID = re.compile(r"^[a-f0-9]{32}$")
GUIDANCE_PROMPT = """Analiza un parser de archivos de laboratorio y su cobertura.
Propón semillas pequeñas no fallantes y tokens para AFL++. No propongas comandos,
URLs, exploits ni cambios de controles. Devuelve únicamente un objeto JSON con:
{"source":"nombre del modelo local", "rationale":"hipótesis comprobable",
 "seeds_base64":["base64 de cada semilla"], "dictionary":["tokens"]}.
Límites: 64 semillas no vacías de hasta 4096 bytes, 64 tokens UTF-8 de hasta 64
bytes, rationale de hasta 2000 caracteres y source de hasta 128 caracteres.
Las semillas que ya fallen se separan como evidencia y no entran al corpus.
Un crash no demuestra una vulnerabilidad desconocida; requiere revisión humana.
"""


def digest(data):
    return hashlib.sha256(data).hexdigest()


def parse_guidance(data):
    if len(data) > MAX_GUIDANCE:
        raise ValueError("Guidance exceeds size limit")
    def unique_object(pairs):
        result = {}
        for key, value in pairs:
            if key in result:
                raise ValueError("Duplicate guidance field")
            result[key] = value
        return result
    obj = json.loads(data, object_pairs_hook=unique_object)
    required = {"source", "rationale", "seeds_base64", "dictionary"}
    if not isinstance(obj, dict) or set(obj) != required:
        raise ValueError("Guidance requires exactly source, rationale, seeds_base64 and dictionary")
    for field, limit in (("source", 128), ("rationale", 2000)):
        if not isinstance(obj[field], str) or not 1 <= len(obj[field]) <= limit:
            raise ValueError("Invalid guidance " + field)
    encoded, tokens = obj["seeds_base64"], obj["dictionary"]
    if not isinstance(encoded, list) or not 1 <= len(encoded) <= 64:
        raise ValueError("Guidance requires 1 to 64 seeds")
    seeds = {}
    for item in encoded:
        if not isinstance(item, str):
            raise ValueError("Seeds must be base64 strings")
        try:
            raw = base64.b64decode(item, validate=True)
        except (ValueError, TypeError) as exc:
            raise ValueError("Invalid base64 seed") from exc
        if not 1 <= len(raw) <= MAX_INPUT:
            raise ValueError("Seed size is outside limits")
        seeds[digest(raw)] = raw
    if not isinstance(tokens, list) or len(tokens) > 64:
        raise ValueError("Guidance accepts at most 64 dictionary tokens")
    for token in tokens:
        if not isinstance(token, str) or not 1 <= len(token.encode("utf-8")) <= 64:
            raise ValueError("Invalid dictionary token")
    return obj, list(seeds.values())


def afl_dictionary(tokens):
    # Hex-escape every byte, including quotes, slashes and control characters.
    return "".join('token%d="%s"\n' % (index, "".join("\\x%02x" % byte for byte in token.encode("utf-8")))
                   for index, token in enumerate(dict.fromkeys(tokens)))


def environment():
    return {"PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
            "HOME": "/tmp", "LANG": "C", "AFL_NO_UI": "1", "AFL_NO_AFFINITY": "1",
            "AFL_SKIP_CPUFREQ": "1", "AFL_I_DONT_CARE_ABOUT_MISSING_CRASHES": "1"}


def replay(path, target=TARGET):
    data = path.read_bytes()
    if not 1 <= len(data) <= MAX_INPUT:
        raise ValueError("Crash input is outside size limits")
    try:
        proc = subprocess.run([str(target), str(path)], timeout=2,
                              stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                              env=environment(), check=False)
    except subprocess.TimeoutExpired:
        return {"sha256": digest(data), "bytes": len(data), "outcome": "timeout"}
    outcome = "crash" if proc.returncode < 0 else ("ok" if proc.returncode == 0 else "error")
    return {"sha256": digest(data), "bytes": len(data), "outcome": outcome,
            "returncode": proc.returncode,
            "signal": -proc.returncode if proc.returncode < 0 else None}


def read_stats(path):
    stats = {}
    if path.is_file():
        for line in path.read_text().splitlines():
            key, sep, value = line.partition(":")
            if sep:
                stats[key.strip()] = value.strip()
    return stats


def save_report(session, report):
    encoded = json.dumps(report, ensure_ascii=False, indent=2) + "\n"
    tmp = session / "report.json.tmp"
    tmp.write_text(encoded)
    tmp.replace(session / "report.json")
    rows = "".join("<tr><td>%s</td><td>%s</td><td>%s</td></tr>" % (
        html.escape(item["sha256"]), html.escape(item["outcome"]), item["bytes"])
        for item in report["findings"])
    page = """<!doctype html><html lang="es"><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>ZERO_day — prueba de laboratorio</title>
<style>body{font:16px system-ui;background:#101923;color:#e5eef9;max-width:1000px;margin:3rem auto;padding:1rem}
table{width:100%%;border-collapse:collapse}td,th{padding:.7rem;border:1px solid #405064;text-align:left}
pre{white-space:pre-wrap;overflow-wrap:anywhere}h1{color:#72d7bc}</style>
<h1>ZERO_day · laboratorio</h1><p>Estado: <strong>%s</strong> · Entradas con crash reproducido: <strong>%d</strong></p>
<p>Programa sintético con un fallo intencional. Los resultados requieren revisión humana;
no son vulnerabilidades desconocidas confirmadas. Los hashes cuentan entradas distintas, no fallos distintos.</p>
<table><tr><th>SHA-256 de la entrada</th><th>Reproducción</th><th>Bytes</th></tr>%s</table>
<h2>Registro de la campaña</h2><pre>%s</pre></html>""" % (
        html.escape(report["status"]), report["reproduced_crashes"], rows, html.escape(encoded))
    (session / "report.html").write_text(page)


def campaign(seconds, guidance_path=None, work=WORK, target=TARGET):
    if not 1 <= seconds <= 600:
        raise ValueError("Duration must be between 1 and 600 seconds")
    if guidance_path:
        with Path(guidance_path).open("rb") as source:
            guidance, seeds = parse_guidance(source.read(MAX_GUIDANCE + 1))
    else:
        guidance, seeds = parse_guidance(json.dumps({"source": "baseline", "rationale": "Baseline corpus",
                                                   "seeds_base64": ["SEVMTE8="], "dictionary": []}).encode())
    if not target.is_file():
        raise ValueError("Instrumented laboratory target is missing; use the zero_day image")
    work.mkdir(parents=True, exist_ok=True)
    session = work / uuid.uuid4().hex
    session.mkdir(mode=0o700)
    corpus = session / "input"
    corpus.mkdir()
    rejected = session / "rejected-seeds"
    rejected.mkdir()
    output = session / "output"
    started = time.time()
    report = {"schema_version": 1, "session_id": session.name, "status": "preparing",
              "target": "synthetic-demo", "target_sha256": digest(target.read_bytes()),
              "guidance": guidance, "guidance_sha256": digest(json.dumps(guidance, sort_keys=True).encode()),
              "requested_seconds": seconds, "started_at": started, "findings": [],
              "reproduced_crashes": 0, "rejected_seeds": [], "stats": {}}
    save_report(session, report)
    proc = None
    try:
        # Check suggested inputs before AFL's dry run. Already crashing inputs
        # are evidence, never attributed to fuzzer discovery.
        accepted = 0
        for seed in seeds:
            path = rejected / digest(seed)
            path.write_bytes(seed)
            result = replay(path, target)
            if result["outcome"] == "ok":
                path.rename(corpus / path.name)
                accepted += 1
            else:
                result["path"] = str(path.relative_to(session))
                report["rejected_seeds"].append(result)
        if not accepted:
            raise ValueError("Guidance has no non-failing seeds")
        (session / "guidance.json").write_text(json.dumps(guidance, ensure_ascii=False, indent=2))
        dictionary = session / "dictionary.txt"
        dictionary.write_text(afl_dictionary(guidance["dictionary"]))
        args = ["afl-fuzz", "-i", str(corpus), "-o", str(output), "-V", str(seconds),
                "-m", "128", "-t", "1000", "-G", str(MAX_INPUT)]
        if guidance["dictionary"]:
            args += ["-x", str(dictionary)]
        args += ["--", str(target), "@@"]
        report["status"] = "running"
        save_report(session, report)
        with (session / "afl.log").open("wb") as log:
            proc = subprocess.Popen(args, stdout=log, stderr=subprocess.STDOUT,
                                    env=environment(), start_new_session=True)
            code = proc.wait(timeout=seconds + 20)
        report["afl_returncode"] = code
        report["stats"] = read_stats(output / "default" / "fuzzer_stats")
        if code != 0 or not report["stats"]:
            raise RuntimeError("AFL++ did not complete successfully; see afl.log")
        seen = set()
        for path in sorted((output / "default" / "crashes").glob("id:*")):
            result = replay(path, target)
            if result["sha256"] in seen:
                continue
            seen.add(result["sha256"])
            result["path"] = str(path.relative_to(session))
            result["review"] = "human_review_required"
            report["findings"].append(result)
        report["reproduced_crashes"] = sum(item["outcome"] == "crash" for item in report["findings"])
        report["status"] = "completed"
    except (KeyboardInterrupt, InterruptedError):
        report["status"] = "cancelled"
    except Exception as exc:
        report["status"] = "failed"
        report["error"] = str(exc)
    finally:
        if proc is not None and proc.poll() is None:
            os.killpg(proc.pid, signal.SIGTERM)
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(proc.pid, signal.SIGKILL)
                proc.wait()
        report["elapsed_seconds"] = round(time.time() - started, 3)
        save_report(session, report)
    return report


def load_session(work, session_id):
    if not SESSION_ID.fullmatch(session_id):
        raise ValueError("Invalid session ID")
    session = work / session_id
    if session.is_symlink() or not session.is_dir():
        raise ValueError("Session not found")
    return session


def main(argv=None):
    parser = argparse.ArgumentParser(description="ZERO_day: campañas AFL++ aisladas de laboratorio")
    commands = parser.add_subparsers(dest="command", required=True)
    run = commands.add_parser("run", help="Ejecutar AFL++ sobre el parser sintético")
    run.add_argument("--seconds", type=int, default=30)
    run.add_argument("--guidance", help="JSON de semillas y diccionario de un modelo local")
    commands.add_parser("prompt", help="Mostrar el contrato para guiar las semillas con un LLM")
    commands.add_parser("list", help="Listar campañas persistidas")
    inspect = commands.add_parser("report", help="Leer el informe JSON de una campaña")
    inspect.add_argument("session")
    reproduce = commands.add_parser("replay", help="Reproducir una entrada guardada")
    reproduce.add_argument("session")
    reproduce.add_argument("sha256")
    args = parser.parse_args(argv)
    try:
        if args.command == "prompt":
            print(GUIDANCE_PROMPT)
            return 0
        if args.command == "list":
            for path in sorted(WORK.glob("*/report.json")):
                obj = json.loads(path.read_text())
                print(obj["session_id"], obj["status"], "crashes=" + str(obj["reproduced_crashes"]))
            return 0
        if args.command == "run":
            result = campaign(args.seconds, args.guidance)
            print(json.dumps(result, ensure_ascii=False, indent=2))
            print("Report: /work/" + result["session_id"] + "/report.html", file=sys.stderr)
            return 0 if result["status"] == "completed" else (130 if result["status"] == "cancelled" else 1)
        session = load_session(WORK, args.session)
        report = json.loads((session / "report.json").read_text())
        if args.command == "report":
            print(json.dumps(report, ensure_ascii=False, indent=2))
            return 0
        if not re.fullmatch(r"[a-f0-9]{64}", args.sha256):
            raise ValueError("Invalid input digest")
        if report["target_sha256"] != digest(TARGET.read_bytes()):
            raise ValueError("Target changed; replay requires the original image")
        for finding in report["findings"] + report["rejected_seeds"]:
            if finding["sha256"] == args.sha256:
                candidate = session / finding["path"]
                if candidate.is_symlink() or not candidate.resolve().is_relative_to(session.resolve()):
                    raise ValueError("Input path is outside the session")
                if digest(candidate.read_bytes()) != args.sha256:
                    raise ValueError("Input digest does not match the saved evidence")
                print(json.dumps(replay(candidate), indent=2))
                return 0
        raise ValueError("Input not found in report")
    except (OSError, ValueError) as exc:
        print("Error: " + str(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    def interrupted(_signum, _frame):
        raise InterruptedError("Campaign interrupted")
    signal.signal(signal.SIGTERM, interrupted)
    sys.exit(main())
