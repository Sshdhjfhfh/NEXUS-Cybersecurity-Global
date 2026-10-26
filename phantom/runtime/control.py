"""Phantom's real transport lifecycle, callable as `nexus phantom ...`."""
import argparse
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import time

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent.parent
ARTIFACTS = HERE / "artifacts"
MODE = ARTIFACTS / "mode.json"


def execute(command, *, capture=False, timeout=180, check=True, env=None):
    return subprocess.run(command, cwd=ROOT, text=True, capture_output=capture, timeout=timeout, check=check, env=env)


def compose(*args, **kwargs):
    path = HERE / "compose.yml"
    if MODE.exists() and json.loads(MODE.read_text()).get("vpn"):
        path = ARTIFACTS / "vpn.compose.json"
    return execute(["docker", "compose", "-f", str(path), *args], **kwargs)


def vpn_compose(path):
    from wireguard import profile
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd) as f:
        st = os.fstat(f.fileno())
        if not stat.S_ISREG(st.st_mode) or st.st_uid != os.getuid() or st.st_mode & 0o077:
            raise ValueError("El perfil WireGuard debe pertenecer a tu usuario y tener permisos 600")
        parsed = profile(f.read(16385))
    # Docker Compose supports JSON natively; no interpolation of keys or hooks.
    base = json.loads(execute(["docker", "compose", "-f", str(HERE / "compose.yml"), "config", "--format", "json"], capture=True).stdout)
    tor = base["services"]["tor"]
    tor.pop("networks", None)
    tor["network_mode"] = "service:vpn"
    tor["depends_on"] = {"vpn": {"condition": "service_healthy"}}
    # Share only the resolver file, not the profile or NET_ADMIN capability.
    resolver = ARTIFACTS / "vpn-resolv.conf"
    resolver.write_text("nameserver " + parsed["dns"] + "\noptions timeout:2 attempts:2\n")
    tor.setdefault("volumes", []).append({"type": "bind", "source": str(resolver), "target": "/etc/resolv.conf", "read_only": True})
    proxy = base["services"]["proxy"]
    proxy["environment"]["PHANTOM_UPSTREAM"] = "vpn:9050"
    base["services"]["vpn"] = {
        "image": "nexus-phantom:local", "user": "0:0", "restart": "unless-stopped",
        "command": ["python3", "/opt/phantom/wireguard.py"],
        "environment": {"PHANTOM_CONTAINER": "1"},
        "networks": {"private": {}, "egress": {}}, "cap_drop": ["ALL"], "cap_add": ["NET_ADMIN", "DAC_READ_SEARCH"],
        "security_opt": ["no-new-privileges:true"], "read_only": True,
        "tmpfs": ["/run:size=4m,noexec,nosuid"], "pids_limit": 64, "mem_limit": "64m",
        "volumes": [{"type": "bind", "source": str(path.absolute()), "target": "/run/secrets/wireguard.conf", "read_only": True},
                    {"type": "bind", "source": str(resolver), "target": "/etc/resolv.conf", "read_only": True}],
        "healthcheck": {"test": ["CMD", "python3", "/opt/phantom/wireguard.py", "health"], "interval": "10s", "timeout": "5s", "retries": 6},
    }
    return base


def check(fault=False):
    report = {"tested_at": time.time(), "mode": "wireguard+tor" if MODE.exists() and json.loads(MODE.read_text()).get("vpn") else "tor", "checks": {}}
    curl = ["exec", "-T", "proxy", "curl", "--silent", "--show-error", "--fail", "--max-time", "45", "--socks5-hostname", "127.0.0.1:1080", "https://check.torproject.org/api/ip"]
    result = compose(*curl, capture=True, check=False, timeout=55)
    try:
        response = json.loads(result.stdout)
    except ValueError:
        response = {}
    report["checks"]["tor_recognized_exit"] = result.returncode == 0 and response.get("IsTor") is True
    report["exit_ip"] = response.get("IP")
    # An internal-only network must have no Internet route, even without SOCKS.
    direct = compose("exec", "-T", "proxy", "python3", "-c", "import socket; socket.create_connection(('1.1.1.1',443),2)", capture=True, check=False)
    report["checks"]["direct_egress_blocked"] = direct.returncode != 0 and "Traceback" in direct.stderr
    ipv6 = compose("exec", "-T", "proxy", "python3", "-c", "import socket; socket.create_connection(('2606:4700:4700::1111',443),2)", capture=True, check=False)
    report["checks"]["direct_ipv6_blocked"] = ipv6.returncode != 0 and "Traceback" in ipv6.stderr
    local_dns = compose("exec", "-T", "proxy", "python3", "-c", "import socket; socket.getaddrinfo('check.torproject.org',443)", capture=True, check=False, timeout=30)
    report["checks"]["local_destination_dns_blocked"] = local_dns.returncode != 0 and "gaierror" in local_dns.stderr
    report["checks"]["fault_test_performed"] = fault
    if fault:
        if not report["checks"]["tor_recognized_exit"]:
            raise RuntimeError("Tor aún no verificó una salida real; no se ejecutó el corte")
        try:
            compose("stop", "tor", capture=True)
            stopped = compose(*curl, capture=True, check=False, timeout=55)
            report["checks"]["tor_failure_blocks_proxy"] = stopped.returncode != 0
        finally:
            compose("up", "-d", "tor", capture=True)
        # Recovery must be observed, not inferred from a successful restart.
        recovered = False
        for _ in range(6):
            recovery = compose(*curl, capture=True, check=False, timeout=55)
            try:
                recovered = recovery.returncode == 0 and json.loads(recovery.stdout).get("IsTor") is True
            except ValueError:
                pass
            if recovered:
                break
            time.sleep(3)
        report["checks"]["tor_recovery_verified"] = recovered
    report["passed"] = all(v for k,v in report["checks"].items() if k != "fault_test_performed")
    report["limits"] = "Valida el proxy aislado y su salida Tor; no certifica anonimato absoluto ni enruta automáticamente Windows o el backend existente. Mixnet propio no está desplegado."
    ARTIFACTS.mkdir(exist_ok=True)
    (ARTIFACTS / "network-check.json").write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps(report, indent=2, ensure_ascii=False))
    return 0 if report["passed"] else 1


def main(argv=None):
    parser = argparse.ArgumentParser(description="Phantom: proxy real Tor y VPN WireGuard aislada")
    parser.add_argument("action", choices=["up", "down", "status", "check", "audit", "vpn-test"], nargs="?", default="status")
    parser.add_argument("--vpn-profile", type=Path, help="Perfil privado WireGuard, solo para up")
    parser.add_argument("--fault", action="store_true", help="Cortar Tor y verificar bloqueo y recuperación")
    args = parser.parse_args(argv)
    if args.vpn_profile and args.action != "up":
        parser.error("--vpn-profile requiere up")
    ARTIFACTS.mkdir(exist_ok=True)
    if args.action == "up":
        new_config = vpn_compose(args.vpn_profile) if args.vpn_profile else None
        print("Compilando el transporte...", flush=True)
        env = dict(os.environ, CGO_ENABLED="0")
        execute(["go", "build", "-trimpath", "-o", str(ARTIFACTS / "phantom-proxy"), "./phantom/runtime/cmd"], env=env)
        compose("build", timeout=600)
        if new_config:
            compose("down")
            (ARTIFACTS / "vpn.compose.json").write_text(json.dumps(new_config, indent=2) + "\n")
            MODE.write_text('{"vpn": true}\n')
        compose("up", "-d", "--wait", "--wait-timeout", "120", timeout=150)
        print("Proxy local: socks5h://127.0.0.1:1080. Verifica la salida con: nexus phantom check", flush=True)
    elif args.action == "down":
        compose("down")
    elif args.action == "status":
        compose("ps")
        print("Proxy: socks5h://127.0.0.1:1080 · DNS remoto · TCP · Sin salida directa de respaldo")
        print("Solo protege tráfico enviado a este proxy. Usa check para verificar la salida actual.")
    elif args.action == "check":
        return check(args.fault)
    elif args.action == "audit":
        return execute([sys.executable, str(ROOT / "terminal" / "phantom_check.py")], check=False, timeout=360).returncode
    elif args.action == "vpn-test":
        return execute([sys.executable, str(HERE / "vpn_test.py")], check=False, timeout=180).returncode
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as exc:
        print("Phantom: " + str(exc), file=sys.stderr)
        sys.exit(1)
