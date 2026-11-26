"""Authenticated loopback entry point for the existing NEXUS session bridge.

An active operator supplies replies using lab/benchmark/relay_queue.py. This
process is not an autonomous model and does not execute the proposed tools.
"""
import argparse
import hmac
from http.server import ThreadingHTTPServer
import os
from pathlib import Path
import secrets
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lab/benchmark'))
from session_bridge import Bridge, response_timeout


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--state', type=Path, default=ROOT / 'build/review/relay')
    parser.add_argument('--port', type=int, default=11435)
    parser.add_argument('--timeout', type=int, default=600)
    args = parser.parse_args()
    if not 1 <= args.port <= 65535:
        parser.error('invalid port')
    os.umask(0o077)
    args.state.mkdir(parents=True, exist_ok=True, mode=0o700)
    queue = args.state / 'queue'
    queue.mkdir(exist_ok=True, mode=0o700)
    key_file = args.state / 'api.key'
    if key_file.is_symlink():
        raise SystemExit('relay key must not be a symlink')
    try:
        with key_file.open('x') as handle:
            handle.write(secrets.token_hex(32))
    except FileExistsError:
        pass
    if key_file.stat().st_mode & 0o077:
        raise SystemExit('relay key must be private (0600)')
    key = key_file.read_text().strip()
    if len(key) != 64:
        raise SystemExit('invalid relay key')

    class LocalBridge(Bridge):
        def authorized(self):
            if not hmac.compare_digest(self.headers.get('Authorization', ''), 'Bearer ' + key):
                self.reply(401, {'error': {'message': 'Session relay authentication required'}})
                return False
            return True

        def do_GET(self):
            if self.authorized():
                super().do_GET()

        def do_POST(self):
            if self.authorized():
                super().do_POST()

    LocalBridge.queue = queue
    LocalBridge.response_seconds = response_timeout(args.timeout)
    server = ThreadingHTTPServer(('127.0.0.1', args.port), LocalBridge)
    server.daemon_threads = True
    print(f'NEXUS session relay: http://127.0.0.1:{args.port}/v1', flush=True)
    print(f'Private queue: {queue}; requires an active operator.', flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == '__main__':
    main()
