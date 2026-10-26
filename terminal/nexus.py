"""Nexus terminal client: existing authenticated API and configured AI agents.

No server keys, database passwords or model API keys are read by this client.
The server retains model selection, tools, permissions and action auditing.
"""
import argparse
import getpass
import json
import os
from pathlib import Path
import re
import shlex
import stat
import subprocess
import sys
import tempfile
import textwrap
import time
import unicodedata
from urllib.error import HTTPError, URLError
from urllib.parse import unquote, urlsplit
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener

ROOT = Path(__file__).resolve().parent.parent
DEFAULT_STATE = Path(os.environ.get("XDG_CONFIG_HOME", Path.home() / ".config")) / "nexus-terminal" / "session.json"
MAX_RESPONSE = 8 * 1024 * 1024
HELP = """NEXUS · Terminal del sistema

Escribe normalmente para hablar con la IA configurada en Nexus.
El agente Auto usa las mismas herramientas y controles que la interfaz web.

  /status                       Estado de Nexus y perfil de IA
  /tasks · /task ID              Tareas y detalle
  /assets · /findings            Activos y hallazgos
  /agents · /skills · /tools     Agentes, skills y herramientas
  /mcp · /logs · /lab            Conexiones, logs y laboratorio
  /models                       Perfiles de IA existentes
  /conversations                Conversaciones existentes
  /new [agente]                 Nueva conversación (por defecto Auto)
  /use ID                       Continuar una conversación existente
  /profile ID|auto              Perfil de esta conversación
  /stop                         Detener la respuesta de la IA
  /api METODO /api/ruta [JSON]   Acceder a cualquier API autorizada
  /zero-day help                Fuzzing, guía local y reproducción
  /phantom [status|up|check|audit]  Proxy Tor real y auditoría
  /phantom check --fault        Probar bloqueo al caer Tor
  /phantom vpn-test             Probar un túnel WireGuard real aislado
  /menu                         Menú de acceso rápido
  /login · /logout              Gestionar la sesión de acceso
  /help · /exit                 Ayuda y salir

Ctrl+C cancela la respuesta en curso. La contraseña solo se pide al iniciar
sesión; después se reutiliza el token guardado hasta que caduque.
"""
MENU = """NEXUS · Acceso rápido
  1  Estado e IA       2  Tareas        3  Activos       4  Hallazgos
  5  Agentes          6  Herramientas  7  Skills        8  Modelos
  9  Conversaciones  10  Laboratorio  11  ZERO_day     12  Phantom

Escribe un número, un comando o directamente tu petición para la IA.
TAB completa los comandos. ayuda muestra todas las opciones; salir termina.
"""
QUICK = {"1": "/status", "2": "/tasks", "3": "/assets", "4": "/findings", "5": "/agents",
         "6": "/tools", "7": "/skills", "8": "/models", "9": "/conversations", "10": "/lab",
         "11": "/zero-day help", "12": "/phantom", "ayuda": "/help", "menu": "/menu",
         "tareas": "/tasks", "activos": "/assets", "hallazgos": "/findings", "estado": "/status"}


def clean(value):
    return "".join(char for char in str(value) if char in "\n\t" or unicodedata.category(char) not in {"Cc", "Cf"})


def display(value):
    print(clean(value if isinstance(value, str) else json.dumps(value, ensure_ascii=False, indent=2)))


def show_resource(route, obj):
    layouts = {
        "/tasks": ("tasks", [("ID", "id"), ("Tarea", "name", "description"), ("Estado", "status")]),
        "/assets": ("assets", [("ID", "id"), ("Tipo", "type", "kind"), ("Activo", "value", "name", "host")]),
        "/findings": ("items", [("ID", "finding_id", "id"), ("Hallazgo", "name", "vulnclass"), ("Gravedad", "severity"), ("Estado", "status")]),
        "/agents": ("agents", [("Agente", "key"), ("Nombre", "name"), ("Rol", "role")]),
        "/skills": ("skills", [("Skill", "name"), ("Descripción", "description")]),
        "/tools": ("tools", [("Herramienta", "key", "name"), ("Descripción", "description")]),
        "/mcp": ("servers", [("ID", "id"), ("MCP", "name")]),
        "/models": ("profiles", [("ID", "id"), ("Perfil", "name"), ("Modelo", "model"), ("Activo", "is_default")]),
        "/conversations": ("conversations", [("ID", "id"), ("Conversación", "title"), ("Agente", "agent_key"), ("En curso", "running")]),
    }
    if route not in layouts:
        display(obj)
        return
    key, columns = layouts[route]
    items = obj if isinstance(obj, list) else obj.get(key) if isinstance(obj, dict) else None
    if not isinstance(items, list):
        display(obj)
        return
    if not items:
        display("No hay registros.")
        return
    rows = []
    for item in items[:30]:
        if not isinstance(item, dict):
            display(obj)
            return
        row = []
        for column in columns:
            value = next((item[key] for key in column[1:] if item.get(key) is not None and item.get(key) != ""), "—")
            value = ("Sí" if value else "No") if isinstance(value, bool) else value
            row.append(textwrap.shorten(clean(value), width=36, placeholder="…"))
        rows.append(row)
    widths = [max(len(column[0]), *(len(row[index]) for row in rows)) for index, column in enumerate(columns)]
    display("  ".join(column[0].ljust(widths[index]) for index, column in enumerate(columns)))
    display("  ".join("─" * width for width in widths))
    for row in rows:
        display("  ".join(value.ljust(widths[index]) for index, value in enumerate(row)))
    display("Mostrando %d de %d registro(s) recibidos. Detalle completo mediante /api." % (min(30, len(items)), len(items)))


class APIError(Exception):
    def __init__(self, status, message):
        self.status = status
        super().__init__(message)


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise ValueError("El servidor intentó redirigir la conexión; las credenciales no se reenviaron.")


def validate_url(url):
    parts = urlsplit(url)
    if (parts.scheme not in {"http", "https"} or not parts.hostname or parts.username or parts.password
            or parts.path not in {"", "/"} or parts.query or parts.fragment):
        raise ValueError("URL de Nexus inválida")
    if parts.scheme == "http" and parts.hostname not in {"localhost", "127.0.0.1", "::1"}:
        raise ValueError("Usa HTTPS para un servidor fuera de loopback")
    return url.rstrip("/")


class Client:
    def __init__(self, url, token="", timeout=20):
        self.url = validate_url(url)
        self.token = token
        self.timeout = timeout
        self.opener = build_opener(ProxyHandler({}), NoRedirect())

    def request(self, method, path, payload=None):
        decoded = unquote(path)
        parts = urlsplit(path)
        if (not path.startswith("/api/") or parts.scheme or parts.netloc or parts.fragment
                or "\\" in decoded or any(part in {".", ".."} for part in urlsplit(decoded).path.split("/"))
                or any(ord(char) < 32 for char in decoded)):
            raise ValueError("La ruta debe permanecer dentro de /api/ del servidor conectado")
        method = method.upper()
        if method not in {"GET", "POST", "PUT", "PATCH", "DELETE"}:
            raise ValueError("Método HTTP no admitido")
        body = json.dumps(payload).encode() if payload is not None else None
        headers = {"Accept": "application/json"}
        if body is not None:
            headers["Content-Type"] = "application/json"
        if self.token:
            headers["Authorization"] = "Bearer " + self.token
        request = Request(self.url + path, data=body, headers=headers, method=method)
        try:
            with self.opener.open(request, timeout=self.timeout) as response:
                raw = response.read(MAX_RESPONSE + 1)
        except HTTPError as exc:
            message = exc.read(4096).decode(errors="replace")
            if exc.code == 401:
                message = "Sesión ausente o caducada. Usa /login."
            raise APIError(exc.code, "HTTP %d: %s" % (exc.code, clean(message))) from exc
        if len(raw) > MAX_RESPONSE:
            raise ValueError("Respuesta demasiado grande; consulta una página más pequeña")
        if not raw:
            return None
        try:
            return json.loads(raw)
        except (ValueError, UnicodeDecodeError):
            return raw.decode(errors="replace")


class SessionStore:
    def __init__(self, path=DEFAULT_STATE):
        self.path = path

    def load(self):
        try:
            fd = os.open(self.path, os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0))
        except FileNotFoundError:
            return {}
        with os.fdopen(fd, "r") as source:
            info = os.fstat(source.fileno())
            if not stat.S_ISREG(info.st_mode) or info.st_mode & 0o077 or info.st_uid != os.getuid() or info.st_size > 65536:
                raise ValueError("El archivo de sesión debe ser privado (0600) y pertenecer a tu usuario")
            obj = json.load(source)
        if not isinstance(obj, dict):
            raise ValueError("Archivo de sesión inválido")
        return obj

    def save(self, obj):
        self.path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        info = self.path.parent.stat()
        if info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise ValueError("El directorio de sesión debe ser privado (0700)")
        if self.path.is_symlink():
            raise ValueError("El archivo de sesión no puede ser un enlace simbólico")
        fd, name = tempfile.mkstemp(prefix=".session-", dir=self.path.parent)
        try:
            with os.fdopen(fd, "w") as target:
                json.dump(obj, target)
                target.flush()
                os.fsync(target.fileno())
            os.replace(name, self.path)
        finally:
            if os.path.exists(name):
                os.unlink(name)


def discover(url=None, saved=None):
    if url:
        candidate = validate_url(url)
        health = Client(candidate, timeout=3).request("GET", "/api/health")
        if not isinstance(health, dict) or health.get("service") != "nexus-ia" or not health.get("ok"):
            raise ValueError("El endpoint no corresponde a Nexus")
        return candidate
    candidates = []
    if saved:
        candidates.append(saved)
    try:
        result = subprocess.run(["docker", "compose", "-f", str(ROOT / "docker-compose.yml"),
                                 "port", "orchestrator", "8787"], capture_output=True, text=True,
                                timeout=4, check=False)
        match = re.fullmatch(r"127\.0\.0\.1:(\d+)\s*", result.stdout)
        if result.returncode == 0 and match:
            candidates.append("http://127.0.0.1:" + match.group(1))
    except (OSError, subprocess.TimeoutExpired):
        pass
    candidates += ["http://127.0.0.1:8789", "http://127.0.0.1:8787"]
    for candidate in dict.fromkeys(candidates):
        try:
            return discover(candidate)
        except (OSError, ValueError, APIError, URLError):
            continue
    raise ValueError("Nexus no responde. Levanta el contenedor o indica --url.")


class Terminal:
    def __init__(self, client, store, saved=None, agent="auto", chat_timeout=600):
        self.client, self.store = client, store
        saved = saved or {}
        same_server = saved.get("url") == client.url
        self.conversation = saved.get("conversation") if same_server else None
        self.agent = saved.get("agent", agent) if same_server else agent
        self.chat_timeout = chat_timeout
        self.zero_controller = None

    def persist(self):
        self.store.save({"url": self.client.url, "token": self.client.token,
                         "conversation": self.conversation, "agent": self.agent})

    def login(self):
        password = getpass.getpass("Contraseña de Nexus: ")
        # The API keeps its legacy login identifier; the visible account is nexus.
        obj = self.client.request("POST", "/api/auth/login", {"username": "ARTEX", "password": password})
        if not isinstance(obj, dict) or not isinstance(obj.get("token"), str):
            raise ValueError("El servidor no devolvió una sesión válida")
        self.client.token = obj["token"]
        self.persist()
        display("Sesión iniciada. Se reutilizará automáticamente.")

    def ensure_conversation(self):
        if self.conversation is None:
            obj = self.client.request("POST", "/api/conversations", {"agent_key": self.agent, "title": "Terminal Nexus"})
            self.conversation = int(obj["id"])
            self.persist()
        return "/api/conversations/" + str(int(self.conversation))

    def status(self):
        health = self.client.request("GET", "/api/health")
        profiles = self.client.request("GET", "/api/llm/profiles").get("profiles") or []
        agents = self.client.request("GET", "/api/agents").get("agents") or []
        active = next((profile for profile in profiles if profile.get("is_default")), None)
        current = next((agent for agent in agents if agent.get("key") == self.agent), {})
        if current.get("llm_profile_id"):
            active = next((profile for profile in profiles if str(profile["id"]) == str(current["llm_profile_id"])), active)
        if self.conversation:
            conversations = self.client.request("GET", "/api/conversations").get("conversations") or []
            selected = next((item for item in conversations if int(item["id"]) == int(self.conversation)), None)
            if selected is None:
                self.conversation = None
                self.persist()
            elif selected.get("llm_profile_id") and not current.get("llm_profile_id"):
                active = next((profile for profile in profiles if str(profile["id"]) == str(selected["llm_profile_id"])), active)
        return {"servidor": self.client.url, "version": health.get("version"), "agente": self.agent,
                "conversacion": self.conversation, "ia": {"perfil": active.get("name"), "modelo": active.get("model")} if active else "El backend no tiene un perfil activo; consulta /models.",
                "seleccion": "La resolución final de modelo, permisos y herramientas corresponde al backend."}

    def chat(self, message):
        if not message.strip() or len(message.encode()) > 128 * 1024:
            raise ValueError("Mensaje vacío o demasiado largo")
        base = self.ensure_conversation()
        before = self.client.request("GET", base + "/messages?since=0")
        if before.get("running"):
            raise ValueError("La conversación sigue trabajando. Espera o usa /stop.")
        cursor = int(before.get("cursor", 0))
        deadline = time.monotonic() + self.chat_timeout
        seen = set()
        last_text = None
        try:
            self.client.request("POST", base + "/messages", {"message": message})
            while True:
                update = self.client.request("GET", base + "/messages?since=" + str(cursor))
                for item in update.get("items") or []:
                    seq = int(item["seq"])
                    if seq in seen:
                        continue
                    seen.add(seq)
                    kind = item.get("kind")
                    if kind in {"text", "result", "error", "intercept_request"}:
                        detail = self.client.request("GET", base + "/messages/" + str(seq)).get("detail")
                        text = detail or item.get("summary", "")
                        if text and text != last_text:
                            display(("IA> " if kind in {"text", "result"} else "Aviso> ") + text)
                            last_text = text
                    elif kind == "tool_use":
                        display("[Herramienta] " + item.get("tool", "") + " · " + item.get("summary", ""))
                    elif item.get("is_error"):
                        display("Error> " + item.get("summary", ""))
                next_cursor = int(update.get("cursor", cursor))
                if next_cursor < cursor:
                    raise ValueError("El cursor de la conversación retrocedió")
                cursor = next_cursor
                # Drain paginated activities even if the backend already finished.
                if not update.get("running") and not update.get("items"):
                    break
                if time.monotonic() >= deadline:
                    raise TimeoutError("La respuesta alcanzó el tiempo límite de la terminal")
                time.sleep(0.4)
        except (KeyboardInterrupt, TimeoutError):
            self.client.request("POST", base + "/stop", {})
            display("Respuesta detenida.")
            return

    def command(self, line):
        line = QUICK.get(line, line)
        if not line.startswith("/"):
            return self.chat(line)
        command, _, rest = line.partition(" ")
        rest = rest.strip()
        routes = {"/tasks": "/api/tasks", "/assets": "/api/assets", "/findings": "/api/exploration/findings",
                  "/agents": "/api/agents", "/skills": "/api/skills", "/tools": "/api/tools",
                  "/mcp": "/api/mcp", "/logs": "/api/logs", "/lab": "/api/lab/status",
                  "/models": "/api/llm/profiles", "/conversations": "/api/conversations"}
        if command in routes and not rest:
            show_resource(command, self.client.request("GET", routes[command]))
        elif command == "/status" and not rest:
            display(self.status())
        elif command == "/help" and not rest:
            display(HELP)
        elif command == "/menu" and not rest:
            display(MENU)
        elif command == "/phantom":
            subprocess.run([sys.executable, str(ROOT / "phantom" / "runtime" / "control.py"), *shlex.split(rest)], check=True)
        elif command == "/login" and not rest:
            self.login()
        elif command == "/logout" and not rest:
            self.client.token = ""
            self.conversation = None
            self.persist()
            display("Sesión local cerrada.")
        elif command == "/task" and rest.isdigit():
            display(self.client.request("GET", "/api/tasks/" + str(int(rest))))
        elif command == "/new":
            key = rest or "auto"
            agents = self.client.request("GET", "/api/agents").get("agents") or []
            if not any(agent.get("key") == key for agent in agents):
                raise ValueError("Agente inexistente; consulta /agents")
            self.agent, self.conversation = key, None
            self.persist()
            display("Nueva conversación con " + key + ". Se usará su IA configurada.")
        elif command == "/use" and rest.isdigit():
            conversations = self.client.request("GET", "/api/conversations").get("conversations") or []
            selected = next((item for item in conversations if int(item["id"]) == int(rest)), None)
            if selected is None:
                raise ValueError("Conversación inexistente; consulta /conversations")
            self.conversation, self.agent = int(rest), selected["agent_key"]
            self.persist()
            display("Conversación seleccionada: " + rest)
        elif command == "/profile" and (rest == "auto" or rest.isdigit()):
            base = self.ensure_conversation()
            self.client.request("PATCH", base + "/profile", {"llm_profile_id": None if rest == "auto" else int(rest)})
            display("Perfil de esta conversación actualizado.")
        elif command == "/stop" and not rest:
            if self.conversation:
                display(self.client.request("POST", "/api/conversations/" + str(self.conversation) + "/stop", {}))
            else:
                display("No hay conversación seleccionada.")
        elif command == "/api":
            parts = rest.split(maxsplit=2)
            if len(parts) < 2:
                raise ValueError("Uso: /api METODO /api/ruta [JSON]")
            display(self.client.request(parts[0], parts[1], json.loads(parts[2]) if len(parts) == 3 else None))
        elif command == "/zero-day":
            sys.path.insert(0, str(ROOT / "zero_day"))
            from control import Controller, wait_for_job
            if self.zero_controller is None:
                self.zero_controller = Controller()
            result = self.zero_controller.command(rest or "help")
            display(result["output"])
            if "job_id" in result:
                wait_for_job(self.zero_controller)
        else:
            raise ValueError("Comando desconocido. Usa /help.")


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    if argv and argv[0] == "phantom":
        return subprocess.call([sys.executable, str(ROOT / "phantom" / "runtime" / "control.py"), *argv[1:]])
    parser = argparse.ArgumentParser(description="Terminal Nexus con conexión automática a su IA")
    parser.add_argument("--url", default=os.environ.get("NEXUS_URL"))
    parser.add_argument("--state", type=Path, default=DEFAULT_STATE)
    parser.add_argument("--command", help="Ejecutar un comando o mensaje y salir")
    parser.add_argument("--doctor", action="store_true", help="Comprobar el backend sin iniciar sesión")
    parser.add_argument("--agent", help="Agente a usar; por defecto Auto o la conversación guardada")
    parser.add_argument("--chat-timeout", type=int, default=600)
    args = parser.parse_args(argv)
    terminal = None
    try:
        if not 1 <= args.chat_timeout <= 1800:
            raise ValueError("El tiempo límite debe estar entre 1 y 1800 segundos")
        if args.command and (args.command == "/phantom" or args.command.startswith("/phantom ")):
            return subprocess.call([sys.executable, str(ROOT / "phantom" / "runtime" / "control.py"), *shlex.split(args.command)[1:]])
        store = SessionStore(args.state)
        saved = store.load()
        url = discover(args.url, saved.get("url"))
        client = Client(url, os.environ.get("NEXUS_TOKEN", saved.get("token", "") if saved.get("url") == url else ""))
        if args.doctor:
            display({"servidor": url, "salud": client.request("GET", "/api/health"),
                     "autenticacion": client.request("GET", "/api/auth/status")})
            return 0
        terminal = Terminal(client, store, saved, args.agent or "auto", args.chat_timeout)
        if args.agent and args.agent != terminal.agent:
            terminal.agent, terminal.conversation = args.agent, None
        if args.command:
            terminal.command(args.command)
            return 0
        display("NEXUS · Terminal del sistema · " + url)
        try:
            display(terminal.status())
        except APIError as exc:
            if exc.status != 401:
                raise
            if not sys.stdin.isatty():
                raise ValueError("Inicia sesión en una terminal interactiva o proporciona NEXUS_TOKEN") from exc
            terminal.login()
            display(terminal.status())
        display("Conexión lista. Escribe a la IA o usa /help. /exit para salir.")
        display(MENU)
        try:
            import readline
            choices = ["/status", "/tasks", "/task ", "/assets", "/findings", "/agents", "/skills", "/tools", "/mcp", "/logs", "/lab", "/models", "/conversations", "/new ", "/use ", "/profile ", "/stop", "/api ", "/zero-day ", "/phantom", "/login", "/logout", "/help", "/menu", "/exit"]
            def complete(text, index):
                matches = [item for item in choices if item.startswith(text)]
                return matches[index] if index < len(matches) else None
            readline.set_completer(complete)
            readline.set_completer_delims(" \t\n")
            readline.parse_and_bind("tab: complete")
        except ImportError:
            pass
        while True:
            try:
                line = input("nexus> ").strip()
                if line in {"/exit", "/quit", "salir"}:
                    break
                if line:
                    terminal.command(line)
            except (EOFError, KeyboardInterrupt):
                display("")
                break
            except (OSError, ValueError, APIError, subprocess.SubprocessError) as exc:
                display("Error: " + str(exc))
        return 0
    except (OSError, ValueError, APIError, subprocess.SubprocessError) as exc:
        display("Error: " + str(exc))
        return 1
    except KeyboardInterrupt:
        display("\nTerminal cerrada.")
        return 130
    finally:
        if terminal and terminal.zero_controller:
            terminal.zero_controller.close()


if __name__ == "__main__":
    sys.exit(main())
