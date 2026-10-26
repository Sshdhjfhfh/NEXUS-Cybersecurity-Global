"""Lab-only relay: this conversation supplies replies; Nexus executes tools.

This measures an assisted integration, not an unattended or blinded benchmark.
Every queued request has an explicit deadline and a terminal lifecycle record.
"""
import json
import os
from pathlib import Path
import select
import socket
import time
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

QUEUE = Path(os.environ.get("NEXUS_BRIDGE_QUEUE", "/queue"))
MODEL = "codex-session-relay"
DEFAULT_RESPONSE_SECONDS = 180


def response_timeout(value):
    seconds = int(value)
    if not 1 <= seconds <= 3600:
        raise ValueError("NEXUS_BRIDGE_RESPONSE_SECONDS must be between 1 and 3600")
    return seconds


def publish(path, value):
    temporary = path.with_name(path.name + "." + uuid.uuid4().hex + ".tmp")
    try:
        fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, "w") as f:
            json.dump(value, f, ensure_ascii=False)
        temporary.replace(path)
    finally:
        temporary.unlink(missing_ok=True)


def validate_message(message, request):
    if not isinstance(message, dict) or message.get("role") != "assistant":
        raise ValueError("invalid response role")
    calls = message.get("tool_calls", [])
    if not isinstance(calls, list):
        raise ValueError("tool_calls must be an array")
    content = message.get("content")
    if content is not None and not isinstance(content, str):
        raise ValueError("assistant content must be text or null")
    if not calls and not content:
        raise ValueError("empty assistant response")
    names = {t["function"]["name"] for t in request.get("tools", [])}
    identifiers = set()
    for call in calls:
        if not isinstance(call, dict) or call.get("type") != "function":
            raise ValueError("invalid tool call")
        ident, function = call.get("id"), call.get("function")
        if not isinstance(ident, str) or not ident or ident in identifiers:
            raise ValueError("tool call IDs must be nonempty and unique")
        identifiers.add(ident)
        if not isinstance(function, dict) or function.get("name") not in names:
            raise ValueError("response names a tool not offered by Nexus")
        arguments = function.get("arguments")
        if not isinstance(arguments, str) or not isinstance(json.loads(arguments), dict):
            raise ValueError("tool arguments must be a JSON object encoded as text")


def disconnected(connection):
    """Detect a canceled HTTP client while no response is available yet."""
    try:
        ready, _, _ = select.select([connection], [], [], 0)
        return bool(ready) and connection.recv(1, socket.MSG_PEEK) == b""
    except OSError:
        return True


class Bridge(BaseHTTPRequestHandler):
    queue = QUEUE
    response_seconds = DEFAULT_RESPONSE_SECONDS
    poll_seconds = .2

    def reply(self, status, payload):
        raw = json.dumps(payload, ensure_ascii=False).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)
        self.wfile.flush()

    def do_GET(self):
        if self.path != "/v1/models":
            self.send_error(404)
            return
        self.reply(200, {"object": "list", "data": [{"id": MODEL, "object": "model", "owned_by": "local-session-relay"}]})

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(404)
            return
        ident, status = None, "failed"
        started = time.monotonic()
        try:
            size = int(self.headers.get("Content-Length", "0"))
            if not 0 < size <= 2 * 1024 * 1024:
                raise ValueError("invalid request size")
            self.connection.settimeout(10)
            request = json.loads(self.rfile.read(size))
            if not isinstance(request, dict) or request.get("model") != MODEL or request.get("stream"):
                raise ValueError("relay requires its named model and non-streaming requests")
            if not isinstance(request.get("messages"), list) or not request["messages"]:
                raise ValueError("messages must be a nonempty array")
            names = [t["function"]["name"] for t in request.get("tools", [])]
            if any(not isinstance(name, str) for name in names):
                raise ValueError("invalid tool names")
            ident = uuid.uuid4().hex
            received_at = time.time()
            deadline = time.monotonic() + self.response_seconds
            publish(self.queue / (ident + ".request.json"), {
                "id": ident, "received_at": received_at,
                "expires_at": received_at + self.response_seconds, "request": request,
            })
            print(json.dumps({"event": "request", "id": ident, "messages": len(request["messages"])}), flush=True)
            response = self.queue / (ident + ".response.json")
            while True:
                if disconnected(self.connection):
                    status = "disconnected"
                    return
                # Expiry takes precedence over a response arriving at the boundary.
                if time.monotonic() >= deadline:
                    status = "expired"
                    self.reply(504, {"error": {"message": "Session relay response deadline exceeded"}})
                    return
                if response.exists():
                    break
                time.sleep(self.poll_seconds)
            message = json.loads(response.read_text())
            validate_message(message, request)
            result = {"id": "chatcmpl-" + ident, "object": "chat.completion", "created": int(time.time()), "model": MODEL,
                      "choices": [{"index": 0, "message": message, "finish_reason": "tool_calls" if message.get("tool_calls") else "stop"}]}
            self.reply(200, result)
            status = "delivered"
        except (ValueError, KeyError, TypeError) as exc:
            status = "invalid_response" if ident else "invalid_request"
            self.reply(502 if ident else 400, {"error": {"message": str(exc)}})
        except (BrokenPipeError, ConnectionResetError, TimeoutError):
            status = "disconnected"
        finally:
            if ident:
                publish(self.queue / (ident + ".done.json"), {
                    "status": status, "at": time.time(),
                    "elapsed_seconds": round(time.monotonic() - started, 3),
                })
                print(json.dumps({"event": "finished", "id": ident, "status": status}), flush=True)

    def log_message(self, *_):
        pass


if __name__ == "__main__":
    os.umask(0o077)
    Bridge.response_seconds = response_timeout(os.environ.get("NEXUS_BRIDGE_RESPONSE_SECONDS", DEFAULT_RESPONSE_SECONDS))
    QUEUE.mkdir(parents=True, exist_ok=True)
    ThreadingHTTPServer(("0.0.0.0", 11434), Bridge).serve_forever()
