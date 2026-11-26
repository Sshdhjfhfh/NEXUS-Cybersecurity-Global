"""Black-box authentication checks for the loopback session relay."""
import json
from pathlib import Path
import socket
import subprocess
import sys
import tempfile
import time
import unittest
from urllib.error import HTTPError, URLError
from urllib.request import Request, ProxyHandler, build_opener


class RelayAuthenticationTest(unittest.TestCase):
    def test_authentication_and_private_state(self):
        with tempfile.TemporaryDirectory() as directory:
            state = Path(directory) / 'relay'
            with socket.socket() as sock:
                sock.bind(('127.0.0.1', 0))
                port = sock.getsockname()[1]
            process = subprocess.Popen([
                sys.executable, str(Path(__file__).with_name('start-session-relay.py')),
                '--state', str(state), '--port', str(port), '--timeout', '5',
            ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            http = build_opener(ProxyHandler({}))
            url = f'http://127.0.0.1:{port}/v1/models'
            try:
                deadline = time.monotonic() + 10
                while True:
                    try:
                        http.open(url, timeout=1)
                    except HTTPError as error:
                        self.assertEqual(error.code, 401)
                        break
                    except URLError:
                        if process.poll() is not None or time.monotonic() > deadline:
                            self.fail('relay failed to start')
                        time.sleep(0.05)
                    else:
                        self.fail('unauthenticated access succeeded')
                key_file = state / 'api.key'
                self.assertEqual(key_file.stat().st_mode & 0o777, 0o600)
                self.assertEqual((state / 'queue').stat().st_mode & 0o777, 0o700)
                key = key_file.read_text().strip()
                request = Request(url, headers={'Authorization': 'Bearer ' + key})
                with http.open(request, timeout=2) as response:
                    models = json.load(response)
                self.assertIn('codex-session-relay', [x['id'] for x in models['data']])
                request = Request(url.replace('/models', '/chat/completions'), data=b'{}',
                                  headers={'Authorization': 'Bearer wrong', 'Content-Type': 'application/json'})
                with self.assertRaises(HTTPError) as denied:
                    http.open(request, timeout=2)
                self.assertEqual(denied.exception.code, 401)
                self.assertEqual(list((state / 'queue').iterdir()), [])
            finally:
                process.terminate()
                process.wait(timeout=5)


if __name__ == '__main__':
    unittest.main()
