"""Fixed local Ollama bridge; no arbitrary hosts, CONNECT, or Internet forwarding."""
import http.client
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Gateway(BaseHTTPRequestHandler):
    def handle_request(self):
        if (self.command, self.path) not in {("GET", "/v1/models"), ("POST", "/v1/chat/completions")}:
            self.send_error(403)
            return
        size = int(self.headers.get("Content-Length", "0"))
        if not 0 <= size <= 2 * 1024 * 1024:
            self.send_error(413)
            return
        conn = http.client.HTTPConnection("host.docker.internal", 11434, timeout=300)
        try:
            conn.request(self.command, self.path, self.rfile.read(size), {"Content-Type": "application/json", "Connection": "close"})
            response = conn.getresponse()
            self.send_response(response.status)
            self.send_header("Content-Type", response.getheader("Content-Type", "application/json"))
            self.send_header("Connection", "close")
            self.end_headers()
            while chunk := response.read1(16384):
                self.wfile.write(chunk)
                self.wfile.flush()
        finally:
            conn.close()
    do_GET = handle_request
    do_POST = handle_request
    def log_message(self, *_):
        pass


ThreadingHTTPServer(("0.0.0.0", 11434), Gateway).serve_forever()
