"""Assign a real task to an isolated Nexus instance using its normal API."""
import json
import os
from pathlib import Path
import secrets
import subprocess
import time
from urllib.request import build_opener, ProxyHandler, Request

HERE = Path(__file__).resolve().parent
OUT = Path(os.environ.get("NEXUS_BENCHMARK_OUT", str(HERE / "artifacts")))
BASE = os.environ.get("NEXUS_BENCHMARK_URL", "http://127.0.0.1:8791")
COMPOSE = ["docker", "compose", "-f", os.environ.get("NEXUS_BENCHMARK_COMPOSE", str(HERE / "compose.yml"))]
if os.environ.get("NEXUS_BENCHMARK_OVERRIDE"):
    COMPOSE += ["-f", os.environ["NEXUS_BENCHMARK_OVERRIDE"]]
if os.environ.get("NEXUS_BENCHMARK_PROJECT"):
    COMPOSE += ["-p", os.environ["NEXUS_BENCHMARK_PROJECT"]]
opener = build_opener(ProxyHandler({}))
token = ""


def api(method, path, data=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    # The agent namespace has no external default route. Use its loopback API
    # through Docker; all requests still use the normal authenticated endpoints.
    code = """import json,sys,urllib.request
p=json.load(sys.stdin)
r=urllib.request.Request('http://127.0.0.1:8787'+p['path'],method=p['method'],headers=p['headers'],data=None if p['data'] is None else json.dumps(p['data']).encode())
with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(r,timeout=30) as response: print(response.read().decode())
"""
    result = subprocess.run(COMPOSE + ["exec", "-T", "nexus", "python3", "-c", code],
        input=json.dumps({"method":method,"path":path,"data":data,"headers":headers}), text=True, capture_output=True, timeout=40)
    if result.returncode:
        raise RuntimeError("Benchmark API failed: " + result.stderr[-600:])
    return json.loads(result.stdout)


def main():
    global token
    OUT.mkdir(parents=True, exist_ok=True)
    session = OUT / "session.json"
    if session.exists():
        token = json.loads(session.read_text())["token"]
    else:
        if api("GET", "/api/auth/status")["initialized"]:
            raise RuntimeError("The benchmark instance was initialized by another session")
        token = api("POST", "/api/auth/init", {"password": secrets.token_urlsafe(24)})["token"]
        fd = os.open(session, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, "w") as f:
            json.dump({"token": token, "url": BASE}, f)
    model = os.environ.get("NEXUS_BENCHMARK_MODEL", "qwen3:4b")
    timeout = int(os.environ.get("NEXUS_BENCHMARK_SECONDS", "300"))
    profile = api("POST", "/api/llm/profiles", {"name": "Evaluación local", "format": "openai", "model": model,
        "base_url": "http://model:11434/v1", "api_key": "local-laboratory-placeholder", "streaming": os.environ.get("NEXUS_BENCHMARK_STREAMING", "true") == "true",
        "max_tokens": int(os.environ.get("NEXUS_BENCHMARK_MAX_TOKENS", "4096")), "context_window_k": 32, "thinking_type": "disabled", "rate_per_second": float(os.environ.get("NEXUS_BENCHMARK_RATE_SECOND", "1")), "rate_per_minute": float(os.environ.get("NEXUS_BENCHMARK_RATE_MINUTE", "10"))})
    api("POST", "/api/llm/profiles/active", {"id": profile["id"]})
    assignment = Path(os.environ.get("NEXUS_BENCHMARK_TASK_FILE", str(HERE / "TASK.md"))).read_text().split("## Evaluación del operador")[0]
    assignment = assignment.replace("5 minutos", str(timeout // 60) + " minutos")
    start = time.time()
    task = api("POST", "/api/tasks", {"name": os.environ.get("NEXUS_BENCHMARK_NAME", "Evaluación real de agentes — laboratorio local"), "description": assignment,
        "goal": os.environ.get("NEXUS_BENCHMARK_GOAL", "Entrega un informe con evidencia HTTP real y correcciones. No basta un plan: ejecuta las comprobaciones GET autorizadas."),
        "llm_profile_ids": [profile["id"]], "timeout_seconds": timeout, "seed_first_intent": os.environ.get("NEXUS_BENCHMARK_SEED_INTENT", "true") == "true", "coverage_enabled": False})
    ident = str(task["id"])
    (OUT / "assignment.json").write_text(json.dumps({"task_id": ident, "model": model, "started_at": start, "assignment": assignment}, ensure_ascii=False, indent=2))
    print("Asignación enviada: tarea " + ident + " · " + model + " · objetivo local http://target:8080", flush=True)
    print("Acceso mediante la API interna del contenedor; evidencias en " + str(OUT), flush=True)
    last = ""
    cursor = 0
    try:
        while time.time() - start < timeout + 30:
            detail = api("GET", "/api/tasks/" + ident)
            state = detail.get("status") or detail.get("task", {}).get("status")
            progress = str(state)
            if progress != last:
                print(f"{int(time.time()-start)}s: {progress}", flush=True)
                last = progress
            (OUT / "task-result.json").write_text(json.dumps(detail, ensure_ascii=False, indent=2))
            activity = api("GET", "/api/exploration/activity?task=" + ident + "&since=" + str(cursor) + "&limit=300")
            with (OUT / "activity.jsonl").open("a") as f:
                for item in activity.get("items", []):
                    f.write(json.dumps(item, ensure_ascii=False) + "\n")
            cursor = activity.get("cursor", cursor)
            if state in {"done", "completed", "failed", "timeout", "cancelled", "stopped", "paused"}:
                break
            time.sleep(5)
    finally:
        detail = api("GET", "/api/tasks/" + ident)
        (OUT / "task-before-cleanup.json").write_text(json.dumps(detail, ensure_ascii=False, indent=2))
        if detail.get("status") not in {"done", "completed", "failed", "timeout", "cancelled", "stopped", "paused"}:
            api("POST", "/api/tasks/" + ident + "/control", {"action": "pause"})
        for service in ["target", "nexus", "model"]:
            logs = subprocess.run(COMPOSE + ["logs", "--no-log-prefix", "--since", str(int(start)), service], capture_output=True, text=True, check=True).stdout
            (OUT / ("target-requests.jsonl" if service == "target" else service + ".log")).write_text(logs)
        for name, path in {
            "activity-final.json": "/api/exploration/activity?task=" + ident + "&since=" + str(cursor) + "&limit=300",
            "intents.json": "/api/exploration/intents?task=" + ident,
            "findings.json": "/api/exploration/findings?task=" + ident,
            "goals.json": "/api/tasks/" + ident + "/goals",
        }.items():
            (OUT / name).write_text(json.dumps(api("GET", path), ensure_ascii=False, indent=2))
        detail = api("GET", "/api/tasks/" + ident)
        (OUT / "task-result.json").write_text(json.dumps(detail, ensure_ascii=False, indent=2))
        print("Prueba terminada. Evidencias guardadas en " + str(OUT), flush=True)


if __name__ == "__main__":
    main()
