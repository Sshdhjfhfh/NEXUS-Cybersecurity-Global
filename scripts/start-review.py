"""Start an isolated, real NEXUS review instance without resuming old tasks.

Uses existing local dependencies. The PostgreSQL cluster, application data,
keys and session relay are kept in build/review and must never be committed.
"""
import json
import os
from pathlib import Path
import socket
import subprocess
import time
from urllib.request import Request, ProxyHandler, build_opener

ROOT = Path(__file__).resolve().parents[1]
STATE = ROOT / 'build/review'
PG = Path('/home/saul/.nexus-dev/pg')
GO = Path('/home/saul/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.linux-amd64/bin/go')
HTTP = build_opener(ProxyHandler({}))


def command(args, **kwargs):
    return subprocess.run([str(a) for a in args], check=True, **kwargs)


def health(url):
    try:
        with HTTP.open(url, timeout=2) as response:
            return response.status == 200
    except Exception:
        return False


def available(port):
    with socket.socket() as sock:
        try:
            sock.bind(('127.0.0.1', port))
            return True
        except OSError:
            return False


def relay_ready():
    key_file = STATE / 'relay/api.key'
    if not key_file.exists():
        return False
    try:
        request = Request('http://127.0.0.1:11435/v1/models', headers={'Authorization':'Bearer ' + key_file.read_text().strip()})
        with HTTP.open(request, timeout=2) as response:
            result = json.load(response)
        return any(model.get('id') == 'codex-session-relay' for model in result.get('data', []))
    except Exception:
        return False


def launch(name, args, env, cwd=ROOT):
    with (STATE / (name + '.log')).open('ab') as output:
        process = subprocess.Popen([str(a) for a in args], cwd=cwd, env=env,
                                   stdin=subprocess.DEVNULL, stdout=output,
                                   stderr=subprocess.STDOUT, start_new_session=True)
    (STATE / (name + '.pid')).write_text(str(process.pid))
    return process


def wait_url(url, seconds, process):
    for _ in range(seconds):
        if health(url):
            return
        if process.poll() is not None:
            raise RuntimeError(f'Service exited: {url}; inspect build/review logs')
        time.sleep(1)
    raise RuntimeError(f'Service did not become healthy: {url}; inspect build/review logs')


def main():
    os.umask(0o077)
    STATE.mkdir(parents=True, exist_ok=True, mode=0o700)
    env = os.environ.copy()
    env['LD_LIBRARY_PATH'] = str(PG / 'lib')
    env['GOPROXY'] = 'off'
    env['GOTOOLCHAIN'] = 'local'
    env['PATH'] = '/home/saul/.nexus-dev/labtools/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin'
    pgdata = STATE / 'pgdata'
    pgsocket = STATE / 'socket'
    pgsocket.mkdir(exist_ok=True, mode=0o700)
    if not (pgdata / 'PG_VERSION').exists():
        command([PG / 'bin/initdb', '-D', pgdata, '-U', 'nexus', '--auth-local=trust', '--auth-host=reject', '--no-locale', '--encoding=UTF8'], env=env, stdout=subprocess.DEVNULL)
    running = subprocess.run([str(PG / 'bin/pg_ctl'), '-D', str(pgdata), 'status'], env=env, stdout=subprocess.DEVNULL).returncode == 0
    if not running:
        # Unix socket directory is private; TCP connections are disabled.
        command([PG / 'bin/pg_ctl', '-D', pgdata, '-l', STATE / 'postgres.log', '-o', f"-k {pgsocket} -h '' -p 55433", '-w', '-t', '30', 'start'], env=env)
    query = command([PG / 'bin/psql', '-h', pgsocket, '-p', '55433', '-U', 'nexus', '-d', 'postgres', '-Atc', "SELECT 1 FROM pg_database WHERE datname='nexus_review'"], env=env, capture_output=True, text=True)
    if query.stdout.strip() != '1':
        command([PG / 'bin/createdb', '-h', pgsocket, '-p', '55433', '-U', 'nexus', 'nexus_review'], env=env)
    env['NEXUS_PG_DSN'] = f'postgresql://nexus@/nexus_review?host={pgsocket}&port=55433&sslmode=disable'
    env['NEXUS_CONFIG'] = str(STATE / 'config.json')
    env['NEXUS_KEY_DIR'] = str(STATE / 'keys')
    env['NEXUS_SKILL_DIR'] = str(STATE / 'skills')
    if not health('http://127.0.0.1:8790/api/health'):
        if not available(8790):
            raise RuntimeError('Port 8790 is occupied; no existing process was stopped')
        command([GO, 'build', '-o', STATE / 'nexus-server', './cmd/nexus'], cwd=ROOT, env=env)
        process = launch('backend', [STATE / 'nexus-server', '-addr', '127.0.0.1:8790', '-data', STATE / 'data', '-proxy', ''], env)
        wait_url('http://127.0.0.1:8790/api/health', 30, process)
    if not relay_ready():
        process = launch('relay', ['/usr/bin/python3', ROOT / 'scripts/start-session-relay.py'], env)
        for _ in range(15):
            if relay_ready():
                break
            if process.poll() is not None:
                raise RuntimeError('Relay exited; inspect build/review/relay.log')
            time.sleep(1)
        else:
            raise RuntimeError('Relay did not become ready; inspect build/review/relay.log')
    if not health('http://127.0.0.1:3000/api/health'):
        if not available(3000):
            raise RuntimeError('Port 3000 is occupied; no existing process was stopped')
        env['NEXUS_API'] = 'http://127.0.0.1:8790'
        env['NEXT_PUBLIC_DEV_BACKEND_PORT'] = '8790'
        env.pop('NEXT_PUBLIC_MOCK', None)
        env.pop('NEXT_EXPORT', None)
        process = launch('web', ['/usr/bin/node', ROOT / 'web/node_modules/next/dist/bin/next', 'dev', '-p', '3000', '-H', '127.0.0.1'], env, ROOT / 'web')
        wait_url('http://127.0.0.1:3000/api/health', 90, process)
    print(json.dumps({'web':'http://127.0.0.1:3000','api':'http://127.0.0.1:8790/api/health','relay':'http://127.0.0.1:11435/v1','state':str(STATE),'database':'nexus_review (isolated, no historic tasks)'}, ensure_ascii=False), flush=True)


if __name__ == '__main__':
    main()
