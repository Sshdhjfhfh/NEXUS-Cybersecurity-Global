from concurrent.futures import ThreadPoolExecutor
import http.client
import json
from pathlib import Path
import socket
import tempfile
import threading
import time
import unittest
from http.server import ThreadingHTTPServer

import session_bridge as bridge
from relay_queue import live_requests, submit


REQUEST = {"model": bridge.MODEL, "stream": False,
           "messages": [{"role": "user", "content": "local probe"}],
           "tools": [{"function": {"name": "Read"}}]}


class QueueTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.queue = Path(self.tmp.name)
        self.ident = "a" * 32
        self.item = {"id": self.ident, "received_at": time.time(),
                     "expires_at": time.time() + 30, "request": REQUEST}
        bridge.publish(self.queue / (self.ident + ".request.json"), self.item)

    def test_submit_is_validated_and_never_overwrites(self):
        with self.assertRaises(ValueError):
            submit(self.queue, self.ident, {"role": "user", "content": "wrong"})
        message = {"role": "assistant", "content": "ready"}
        submit(self.queue, self.ident, message)
        with self.assertRaises(ValueError):
            submit(self.queue, self.ident, {"role": "assistant", "content": "replacement"})
        saved = self.queue / (self.ident + ".response.json")
        self.assertEqual(json.loads(saved.read_text()), message)
        self.assertEqual(saved.stat().st_mode & 0o777, 0o600)

    def test_expired_and_finished_requests_are_not_live(self):
        self.assertEqual(list(live_requests(self.queue, now=self.item["expires_at"])), [])
        bridge.publish(self.queue / (self.ident + ".done.json"), {"status": "disconnected"})
        self.assertEqual(list(live_requests(self.queue)), [])
        with self.assertRaises(ValueError):
            submit(self.queue, self.ident, {"role": "assistant", "content": "late"})

    def test_legacy_request_deadline(self):
        del self.item["expires_at"]
        self.item["received_at"] = time.time() - 181
        bridge.publish(self.queue / (self.ident + ".request.json"), self.item)
        self.assertEqual(list(live_requests(self.queue)), [])

    def test_tool_response_validation(self):
        def message(name="Read", args='{}', ident="call1"):
            return {"role": "assistant", "content": None, "tool_calls": [
                {"id": ident, "type": "function", "function": {"name": name, "arguments": args}}]}
        bridge.validate_message(message(), REQUEST)
        for value in (message(name="Missing"), message(args='[]'), message(args='{'),
                      message(ident=""), {"role": "assistant", "content": None}):
            with self.subTest(value=value), self.assertRaises((ValueError, TypeError)):
                bridge.validate_message(value, REQUEST)
        duplicate = message()
        duplicate["tool_calls"] *= 2
        with self.assertRaises(ValueError):
            bridge.validate_message(duplicate, REQUEST)

    def test_timeout_configuration_is_bounded(self):
        self.assertEqual(bridge.response_timeout("600"), 600)
        for value in ("0", "-1", "3601", "infinite"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                bridge.response_timeout(value)


class LifecycleTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.queue = Path(self.tmp.name)
        handler = type("TestBridge", (bridge.Bridge,), {
            "queue": self.queue, "response_seconds": 2, "poll_seconds": .01})
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.executor = ThreadPoolExecutor(max_workers=1)
        self.addCleanup(self.cleanup)

    def cleanup(self):
        self.executor.shutdown(wait=True)
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()
        self.tmp.cleanup()

    def post(self, value=REQUEST):
        conn = http.client.HTTPConnection(*self.server.server_address, timeout=4)
        try:
            conn.request("POST", "/v1/chat/completions", json.dumps(value), {"Content-Type": "application/json"})
            res = conn.getresponse()
            return res.status, json.loads(res.read())
        finally:
            conn.close()

    def wait_file(self, pattern):
        until = time.monotonic() + 3
        while time.monotonic() < until:
            paths = list(self.queue.glob(pattern))
            if paths:
                return paths[0]
            time.sleep(.005)
        self.fail("missing " + pattern)

    def test_delivered_response_has_deadline_and_terminal_record(self):
        future = self.executor.submit(self.post)
        request = json.loads(self.wait_file("*.request.json").read_text())
        self.assertGreater(request["expires_at"], request["received_at"])
        submit(self.queue, request["id"], {"role": "assistant", "content": "ready"})
        self.assertEqual(future.result()[0], 200)
        self.assertEqual(json.loads(self.wait_file("*.done.json").read_text())["status"], "delivered")

    def test_expiry_is_terminal(self):
        self.server.RequestHandlerClass.response_seconds = .05
        status, _ = self.post()
        self.assertEqual(status, 504)
        self.assertEqual(json.loads(self.wait_file("*.done.json").read_text())["status"], "expired")
        self.assertEqual(list(live_requests(self.queue)), [])

    def test_invalid_reply_is_terminal(self):
        future = self.executor.submit(self.post)
        request = json.loads(self.wait_file("*.request.json").read_text())
        bridge.publish(self.queue / (request["id"] + ".response.json"), {"role": "user"})
        self.assertEqual(future.result()[0], 502)
        self.assertEqual(json.loads(self.wait_file("*.done.json").read_text())["status"], "invalid_response")

    def test_client_cancellation_does_not_wait_for_expiry(self):
        conn = socket.create_connection(self.server.server_address, timeout=2)
        raw = json.dumps(REQUEST).encode()
        conn.sendall(b"POST /v1/chat/completions HTTP/1.1\r\nHost: localhost\r\nContent-Length: "
                     + str(len(raw)).encode() + b"\r\n\r\n" + raw)
        self.wait_file("*.request.json")
        conn.close()
        terminal = json.loads(self.wait_file("*.done.json").read_text())
        self.assertEqual(terminal["status"], "disconnected")
        self.assertLess(terminal["elapsed_seconds"], 1.5)

    def test_bad_request_does_not_create_queue_entry(self):
        self.assertEqual(self.post([])[0], 400)
        self.assertEqual(list(self.queue.iterdir()), [])


if __name__ == "__main__":
    unittest.main()
