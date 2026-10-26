import contextlib
import io
import json
import os
from pathlib import Path
import re
import tempfile
import unittest
from unittest.mock import patch

import nexus


class TransportTests(unittest.TestCase):
    def test_only_local_http_or_verified_https_origins(self):
        for url in ["http://example.com", "http://user:password@localhost:8787", "http://localhost/path", "file:///etc/passwd", "http://localhost?token=secret"]:
            with self.subTest(url=url), self.assertRaises(ValueError):
                nexus.Client(url)
        self.assertEqual(nexus.validate_url("http://127.0.0.1:8789/"), "http://127.0.0.1:8789")

    def test_api_paths_cannot_escape_or_forward_token(self):
        client = nexus.Client("http://127.0.0.1:8789", "session-token")
        for path in ["https://example.com/api/tasks", "//example.com/api/tasks", "/api/../secrets", "/api/%2e%2e/secrets", "/api/path\\outside", "/api/%0aHeader"]:
            with self.subTest(path=path), self.assertRaises(ValueError):
                client.request("GET", path)
        with patch.object(client.opener, "open", return_value=io.BytesIO(b'{"ok":true}')) as send:
            self.assertEqual(client.request("POST", "/api/tasks", {"description": "A title with spaces"}), {"ok": True})
            request = send.call_args.args[0]
            self.assertEqual(request.get_header("Authorization"), "Bearer session-token")
            self.assertEqual(json.loads(request.data)["description"], "A title with spaces")

    def test_terminal_control_sequences_are_removed(self):
        self.assertNotIn("\x1b", nexus.clean("model\x1b]52;c;clipboard\x07"))
        self.assertNotIn("\r", nexus.clean("overwrite\rnext"))


class SessionTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name) / "private" / "session.json"
        self.store = nexus.SessionStore(self.path)

    def test_private_session_survives_restart_without_password(self):
        obj = {"url": "http://127.0.0.1:8789", "token": "test-token", "conversation": 9}
        self.store.save(obj)
        self.assertEqual(nexus.SessionStore(self.path).load(), obj)
        self.assertEqual(self.path.stat().st_mode & 0o777, 0o600)
        self.assertEqual(self.path.parent.stat().st_mode & 0o777, 0o700)
        self.assertNotIn("password", self.path.read_text())

    def test_public_or_symlink_session_is_rejected(self):
        self.store.save({"token": "test"})
        self.path.chmod(0o644)
        with self.assertRaises(ValueError):
            self.store.load()
        self.path.unlink()
        other = Path(self.temp.name) / "other"
        other.write_text("secret")
        self.path.symlink_to(other)
        with self.assertRaises(OSError):
            self.store.load()
        with self.assertRaises(ValueError):
            self.store.save({"token": "test"})

    def test_session_not_reused_for_a_different_server(self):
        terminal = nexus.Terminal(nexus.Client("http://127.0.0.1:8789"), self.store,
                                  {"url": "https://other.example", "conversation": 1, "agent": "other"})
        self.assertIsNone(terminal.conversation)
        self.assertEqual(terminal.agent, "auto")


class FakeClient:
    url = "http://127.0.0.1:8789"
    token = "fixture-session"

    def __init__(self):
        self.calls = []
        self.responses = {}

    def request(self, method, path, payload=None):
        self.calls.append((method, path, payload))
        obj = self.responses[(method, path)]
        if isinstance(obj, list):
            obj = obj.pop(0)
        if isinstance(obj, BaseException):
            raise obj
        return obj


class TerminalTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.client = FakeClient()
        self.terminal = nexus.Terminal(self.client, nexus.SessionStore(Path(self.temp.name) / "session.json"))

    def test_chat_uses_backend_agent_and_drains_complete_response(self):
        self.client.responses = {
            ("POST", "/api/conversations"): {"id": 12},
            ("GET", "/api/conversations/12/messages?since=0"): [
                {"cursor": 0, "running": False, "items": []},
                {"cursor": 2, "running": False, "items": [{"seq": 2, "kind": "text", "summary": "short"}]}],
            ("POST", "/api/conversations/12/messages"): {"status": "accepted"},
            ("GET", "/api/conversations/12/messages/2"): {"detail": "Respuesta completa del modelo configurado"},
            ("GET", "/api/conversations/12/messages?since=2"): {"cursor": 2, "running": False, "items": []},
        }
        output = io.StringIO()
        with contextlib.redirect_stdout(output), patch.object(nexus.time, "sleep"):
            self.terminal.chat("Muéstrame el estado")
        self.assertIn("Respuesta completa", output.getvalue())
        created = next(call for call in self.client.calls if call[:2] == ("POST", "/api/conversations"))
        self.assertEqual(created[2]["agent_key"], "auto")
        self.assertNotIn("llm_profile_id", created[2])
        self.assertEqual(self.terminal.conversation, 12)

    def test_interrupt_stops_the_same_backend_conversation(self):
        self.terminal.conversation = 9
        self.client.responses = {
            ("GET", "/api/conversations/9/messages?since=0"): [
                {"cursor": 0, "running": False, "items": []}, KeyboardInterrupt()],
            ("POST", "/api/conversations/9/messages"): {"status": "accepted"},
            ("POST", "/api/conversations/9/stop"): {"status": "stopped"},
        }
        with contextlib.redirect_stdout(io.StringIO()):
            self.terminal.chat("hola")
        self.assertIn(("POST", "/api/conversations/9/stop", {}), self.client.calls)

    def test_generic_api_preserves_json_and_system_access(self):
        self.client.responses[("PATCH", "/api/agents/auto")] = {"ok": True}
        with contextlib.redirect_stdout(io.StringIO()):
            self.terminal.command('/api PATCH /api/agents/auto {"name":"A name with spaces"}')
        self.assertEqual(self.client.calls[-1][2], {"name": "A name with spaces"})

    def test_menu_number_uses_system_api(self):
        self.client.responses[("GET", "/api/tasks")] = {"tasks": []}
        with contextlib.redirect_stdout(io.StringIO()):
            self.terminal.command("2")
        self.assertEqual(self.client.calls[-1][:2], ("GET", "/api/tasks"))

    def test_shortcuts_match_routes_registered_by_the_real_server(self):
        routes = set()
        for source in (nexus.ROOT / "orchestrator" / "server").glob("*.go"):
            if not source.name.endswith("_test.go"):
                routes.update(re.findall(r'"GET (/api/[^"\s]+)"', source.read_text()))
        with patch.object(self.client, "request", return_value={}) as request, contextlib.redirect_stdout(io.StringIO()):
            for shortcut in ["/tasks", "/assets", "/findings", "/agents", "/skills", "/tools", "/mcp", "/logs", "/lab", "/models", "/conversations"]:
                self.terminal.command(shortcut)
                self.assertIn(request.call_args.args[1], routes, shortcut)

    def test_status_honors_agent_profile_before_conversation(self):
        self.terminal.conversation = 9
        self.client.responses = {
            ("GET", "/api/health"): {"version": "test"},
            ("GET", "/api/llm/profiles"): {"profiles": [
                {"id": "1", "name": "global", "model": "model1", "is_default": True},
                {"id": "2", "name": "bound", "model": "model2"}]},
            ("GET", "/api/agents"): {"agents": [{"key": "auto", "llm_profile_id": 2}]},
            ("GET", "/api/conversations"): {"conversations": [{"id": 9, "llm_profile_id": 1}]},
        }
        self.assertEqual(self.terminal.status()["ia"]["modelo"], "model2")


if __name__ == "__main__":
    unittest.main()
