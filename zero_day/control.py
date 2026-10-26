"""Host-side terminal controls for the isolated ZERO_day demo."""
import argparse
import copy
import json
import os
from pathlib import Path
import re
import shlex
import signal
import subprocess
import sys
import threading
import time
import uuid

from campaign import SESSION_ID, load_session, parse_guidance

ROOT = Path(__file__).resolve().parent
ARTIFACTS = ROOT / "artifacts"
HELP = """Comandos ZERO_day:
  run [segundos] [example|guided|baseline]  Ejecutar AFL++ (1–600 s)
  guide [modelo]                         Generar semillas con Ollama local
  list                                   Listar campañas
  report IDENTIFICADOR                    Ver un informe JSON
  replay IDENTIFICADOR SHA256             Reproducir una entrada
  status                                 Ver el trabajo actual
  cancel                                 Cancelar el trabajo actual
  help                                   Mostrar ayuda
  exit                                   Salir de la terminal local
La terminal controla este módulo; no ejecuta comandos del sistema.
"""


def read_report(session_id, root=ARTIFACTS):
    session = load_session(root, session_id)
    path = session / "report.json"
    if path.is_symlink() or path.stat().st_size > 1024 * 1024:
        raise ValueError("Invalid report file")
    obj = json.loads(path.read_text())
    if not isinstance(obj, dict) or obj.get("session_id") != session_id:
        raise ValueError("Report identity mismatch")
    return obj


def reports(root=ARTIFACTS):
    if not root.exists():
        return []
    result = []
    paths = sorted(root.iterdir(), key=lambda path: path.stat().st_mtime, reverse=True)
    for path in paths:
        if SESSION_ID.fullmatch(path.name):
            try:
                result.append(read_report(path.name, root))
            except (ValueError, OSError):
                continue
        if len(result) == 100:
            break
    return sorted(result, key=lambda obj: obj.get("started_at", 0), reverse=True)


def tail(path, limit=24000):
    if not path.is_file() or path.is_symlink():
        return ""
    with path.open("rb") as source:
        source.seek(max(0, path.stat().st_size - limit))
        return source.read(limit).decode(errors="replace")


class Controller:
    def __init__(self, root=ROOT):
        self.root = root
        self.artifacts = root / "artifacts"
        self.artifacts.mkdir(parents=True, exist_ok=True)
        self.lock = threading.RLock()
        self.job = None
        self.process = None
        self.thread = None
        self.stopping = False

    def state(self):
        runs = reports(self.artifacts)
        with self.lock:
            job = copy.deepcopy(self.job)
        text = tail(self.artifacts / "control.log")
        if job and job["kind"] == "run":
            active = next((item for item in runs if item.get("started_at", 0) >= job["started_at"] - 1), None)
            if active:
                text = tail(self.artifacts / active["session_id"] / "afl.log") + "\n" + text
        guidance = None
        path = self.artifacts / "guidance.json"
        if path.is_file() and not path.is_symlink():
            try:
                with path.open("rb") as source:
                    guidance, _ = parse_guidance(source.read(1024 * 1024 + 1))
            except (ValueError, OSError):
                pass
        return {"job": job, "runs": runs, "guidance": guidance, "log": text}

    def command(self, command):
        if not isinstance(command, str) or not 1 <= len(command) <= 512:
            raise ValueError("Invalid command length")
        args = shlex.split(command)
        if not args:
            raise ValueError("Command is empty")
        name = args[0]
        if name == "help" and len(args) == 1:
            return {"output": HELP}
        if name == "status" and len(args) == 1:
            return {"output": json.dumps(self.state()["job"], ensure_ascii=False, indent=2)}
        if name == "list" and len(args) == 1:
            items = reports(self.artifacts)
            return {"output": "\n".join("%s %s crashes=%s" % (item["session_id"], item["status"], item["reproduced_crashes"]) for item in items) or "Sin campañas todavía."}
        if name == "report" and len(args) == 2:
            return {"output": json.dumps(read_report(args[1], self.artifacts), ensure_ascii=False, indent=2)}
        if name == "cancel" and len(args) == 1:
            return {"output": self.cancel()}
        container = None
        identifier = uuid.uuid4().hex
        compose = ["docker", "compose", "-f", str(self.root / "compose.yml")]
        if name == "run" and 1 <= len(args) <= 3:
            seconds = int(args[1]) if len(args) >= 2 else 30
            mode = args[2] if len(args) == 3 else "example"
            if not 1 <= seconds <= 600 or mode not in {"example", "guided", "baseline"}:
                raise ValueError("Use run [1–600] [example|guided|baseline]")
            if mode == "guided":
                with (self.artifacts / "guidance.json").open("rb") as source:
                    parse_guidance(source.read(1024 * 1024 + 1))
            container = "nexus-zero-day-" + identifier
            argv = compose + ["run", "--rm", "-T", "--name", container, "fuzz", "run", "--seconds", str(seconds)]
            if mode != "baseline":
                argv += ["--guidance", "/work/guidance.json" if mode == "guided" else "/opt/zero-day/guidance.example.json"]
        elif name == "guide" and len(args) <= 2:
            model = args[1] if len(args) == 2 else "qwen3:4b"
            if not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9._:/-]{0,127}", model):
                raise ValueError("Invalid local model name")
            argv = [sys.executable, str(self.root / "guide.py"), "--model", model,
                    "--output", str(self.artifacts / "guidance.json")]
        elif name == "replay" and len(args) == 3:
            read_report(args[1], self.artifacts)
            if not re.fullmatch(r"[a-f0-9]{64}", args[2]):
                raise ValueError("Invalid input digest")
            container = "nexus-zero-day-" + identifier
            argv = compose + ["run", "--rm", "-T", "--name", container, "fuzz", "replay", args[1], args[2]]
        else:
            raise ValueError("Comando desconocido. Escribe help.")
        with self.lock:
            if self.stopping:
                raise ValueError("Controller is stopping")
            if self.job and self.job["status"] in {"running", "cancelling"}:
                raise ValueError("Ya hay un trabajo en curso. Espera o usa cancel.")
            self.job = {"id": identifier, "kind": name, "command": command, "status": "running",
                        "started_at": time.time(), "container": container}
            (self.artifacts / "control.log").write_text("")
            self.thread = threading.Thread(target=self._execute, args=(argv, identifier), daemon=True)
            self.thread.start()
            return {"output": "Trabajo iniciado: " + command, "job_id": identifier}

    def _execute(self, argv, identifier):
        code, error = 1, None
        try:
            with (self.artifacts / "control.log").open("wb") as log:
                with self.lock:
                    if self.job["status"] == "cancelling":
                        self.job.update(status="cancelled", finished_at=time.time(), returncode=130)
                        return
                    self.process = subprocess.Popen(argv, stdout=log, stderr=subprocess.STDOUT,
                                                    start_new_session=True, cwd=self.root)
                    process = self.process
                code = process.wait(timeout=650)
        except Exception as exc:
            error = str(exc)
            self._terminate()
        finally:
            with self.lock:
                if self.job["id"] == identifier:
                    cancelled = self.job["status"] in {"cancelling", "cancelled"}
                    self.job.update(status="cancelled" if cancelled else ("completed" if code == 0 else "failed"),
                                    returncode=130 if cancelled else code, finished_at=time.time())
                    if error:
                        self.job["error"] = error
                    self.process = None

    def _terminate(self):
        with self.lock:
            process = self.process
            container = self.job.get("container") if self.job else None
        if container and process and process.poll() is None:
            try:
                subprocess.run(["docker", "stop", "--timeout", "5", container],
                               capture_output=True, timeout=15, check=False)
            except (OSError, subprocess.TimeoutExpired):
                pass
        if process and process.poll() is None:
            os.killpg(process.pid, signal.SIGTERM)
            try:
                process.wait(timeout=8)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()

    def cancel(self):
        with self.lock:
            if not self.job or self.job["status"] not in {"running", "cancelling"}:
                return "No hay ningún trabajo activo."
            self.job["status"] = "cancelling"
        self._terminate()
        return "Cancelación solicitada."

    def close(self):
        with self.lock:
            self.stopping = True
        self.cancel()
        if self.thread:
            self.thread.join(timeout=20)


def wait_for_job(controller, verbose=False):
    previous = ""
    last_tick = -1
    try:
        while True:
            state = controller.state()
            text = state["log"]
            if verbose and text != previous:
                print(text[len(previous):] if text.startswith(previous) else text, end="", flush=True)
                previous = text
            if state["job"]["status"] not in {"running", "cancelling"}:
                print("\nEstado: " + state["job"]["status"])
                if state["job"]["kind"] == "run":
                    run = next((item for item in state["runs"] if item.get("started_at", 0) >= state["job"]["started_at"] - 1), None)
                    if run:
                        print("Sesión: " + run["session_id"])
                        print("Ejecuciones: %s · Crashes reproducidos: %s" % (run.get("stats", {}).get("execs_done", "0"), run.get("reproduced_crashes", 0)))
                        print("Informe: " + str(controller.artifacts / run["session_id"] / "report.html"))
                if state["job"]["status"] == "failed":
                    print(state["log"][-2000:])
                return 0 if state["job"]["status"] == "completed" else 1
            elapsed = int(time.time() - state["job"]["started_at"])
            if not verbose and elapsed != last_tick:
                print("\rZERO_day · %s · %ss" % (state["job"]["kind"], elapsed), end="", flush=True)
                last_tick = elapsed
            time.sleep(0.5)
    except KeyboardInterrupt:
        print("\n" + controller.cancel())
        if controller.thread:
            controller.thread.join(timeout=20)
        return 130


def main(argv=None):
    parser = argparse.ArgumentParser(description="ZERO_day: terminal local")
    parser.add_argument("command", nargs="?", default="shell", help="shell o un comando ZERO_day entre comillas")
    args = parser.parse_args(argv)
    controller = Controller()
    try:
        if args.command != "shell":
            result = controller.command(args.command)
            print(result["output"])
            return wait_for_job(controller) if "job_id" in result else 0
        print(HELP)
        while True:
            try:
                command = input("ZERO_day> ").strip()
                if command in {"exit", "quit"}:
                    break
                if not command:
                    continue
                result = controller.command(command)
                print(result["output"])
                if "job_id" in result:
                    wait_for_job(controller)
            except (EOFError, KeyboardInterrupt):
                print()
                break
            except (OSError, ValueError) as exc:
                print("Error: " + str(exc))
        return 0
    except (OSError, ValueError) as exc:
        print("Error: " + str(exc), file=sys.stderr)
        return 1
    finally:
        controller.close()


if __name__ == "__main__":
    sys.exit(main())
