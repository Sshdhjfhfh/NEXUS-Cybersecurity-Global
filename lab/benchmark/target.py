"""Owned, read-only HTTP assessment target with synthetic records only."""
import json
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlsplit


class Target(BaseHTTPRequestHandler):
    server_version = "NexusTraining/1.0"
    def do_GET(self):
        path = urlsplit(self.path).path
        routes = {
            "/": ("text/html", '<h1>Nexus local assessment</h1><a href="/status">Status</a><a href="/api/reports/1">Report</a><a href="/robots.txt">Robots</a>'),
            "/status": ("application/json", json.dumps({"service": "training", "data": "synthetic", "write_operations": False})),
            "/robots.txt": ("text/plain", "User-agent: *\nDisallow: /debug/config\n"),
            "/debug/config": ("application/json", json.dumps({"debug": True, "database_password": "DEMO-ONLY-NOT-A-REAL-CREDENTIAL", "internal_service": "training-database", "environment": "local-benchmark"})),
            "/api/reports/1": ("application/json", json.dumps({"id": 1, "owner": "alice", "classification": "private", "content": "SYNTHETIC-ALICE-REPORT"})),
            "/api/reports/2": ("application/json", json.dumps({"id": 2, "owner": "bob", "classification": "private", "content": "SYNTHETIC-BOB-REPORT"})),
        }
        kind, body = routes.get(path, ("text/plain", "Not found"))
        encoded = body.encode()
        self.send_response(200 if path in routes else 404)
        self.send_header("Content-Type", kind)
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)
    def log_request(self, code="-", size="-"):
        print(json.dumps({"time": time.time(), "method": self.command, "path": self.path, "status": code}), flush=True)
    def log_message(self, *_):
        pass


ThreadingHTTPServer(("0.0.0.0", 8080), Target).serve_forever()
