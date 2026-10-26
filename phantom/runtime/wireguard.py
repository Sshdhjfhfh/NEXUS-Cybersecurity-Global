"""WireGuard gateway for a dedicated Docker network namespace, never the host.

Only a declarative, single-peer IPv4 full-tunnel profile is accepted. No hooks.
IPv6 is blocked until a dual-stack routing and leak suite is implemented.
"""
import base64
import configparser
import ipaddress
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time


def key(value):
    try:
        if len(base64.b64decode(value, validate=True)) != 32:
            raise ValueError()
    except ValueError:
        raise ValueError("Invalid WireGuard key") from None
    return value


def profile(text):
    if len(text.encode()) > 16384:
        raise ValueError("Profile too large")
    cfg = configparser.ConfigParser(interpolation=None, strict=True, inline_comment_prefixes=("#", ";"))
    cfg.optionxform = str
    cfg.read_string(text)
    if cfg.defaults() or set(cfg.sections()) != {"Interface", "Peer"}:
        raise ValueError("Exactly one Interface and one Peer are required")
    interface, peer = cfg["Interface"], cfg["Peer"]
    if set(interface) - {"PrivateKey", "Address", "DNS", "MTU"} or set(peer) - {"PublicKey", "PresharedKey", "Endpoint", "AllowedIPs", "PersistentKeepalive"}:
        raise ValueError("Unsupported profile fields; hooks, custom routes and scripts are forbidden")
    addresses = [ipaddress.ip_interface(value.strip()) for value in interface["Address"].split(",")]
    v4 = [value for value in addresses if value.version == 4]
    if len(v4) != 1 or v4[0].ip.is_loopback or v4[0].ip.is_unspecified or v4[0].ip.is_multicast:
        raise ValueError("One IPv4 tunnel address is required")
    dns = ipaddress.IPv4Address(interface["DNS"].split(",")[0].strip())
    if dns.is_loopback or dns.is_unspecified or dns.is_multicast:
        raise ValueError("A DNS resolver reachable through the tunnel is required")
    endpoint, port = peer["Endpoint"].rsplit(":", 1)
    endpoint = ipaddress.IPv4Address(endpoint)
    if endpoint.is_loopback or endpoint.is_unspecified or endpoint.is_multicast or not 1 <= int(port) <= 65535:
        raise ValueError("A literal IPv4 endpoint and valid port are required; no bootstrap DNS")
    allowed = {str(ipaddress.ip_network(value.strip(), strict=True)) for value in peer["AllowedIPs"].split(",")}
    if "0.0.0.0/0" not in allowed or allowed - {"0.0.0.0/0", "::/0"}:
        raise ValueError("Only full-tunnel AllowedIPs are supported")
    mtu = int(interface.get("MTU", "1420"))
    keepalive = int(peer.get("PersistentKeepalive", "25"))
    if not 1280 <= mtu <= 1420 or not 1 <= keepalive <= 120:
        raise ValueError("Unsupported MTU or keepalive")
    stripped = "[Interface]\nPrivateKey = " + key(interface["PrivateKey"]) + "\n[Peer]\nPublicKey = " + key(peer["PublicKey"])
    if "PresharedKey" in peer:
        stripped += "\nPresharedKey = " + key(peer["PresharedKey"])
    stripped += f"\nEndpoint = {endpoint}:{int(port)}\nAllowedIPs = 0.0.0.0/0\nPersistentKeepalive = {keepalive}\n"
    return {"address": str(v4[0]), "dns": str(dns), "endpoint": str(endpoint), "port": int(port), "mtu": mtu, "config": stripped}


def run(*args, data=None):
    result = subprocess.run(args, input=data, text=True, capture_output=True, timeout=15)
    if result.returncode:
        # Never echo profile contents or secret key arguments.
        raise RuntimeError("Gateway command failed: " + " ".join(args[:2]))
    return result.stdout


def firewall(endpoint, port):
    # Established only permits responses to already admitted flows. No forwarding.
    rules = f"""*filter
:INPUT DROP [0:0]
:FORWARD DROP [0:0]
:OUTPUT DROP [0:0]
-A INPUT -i lo -j ACCEPT
-A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
-A INPUT -p tcp --dport 9050 -j ACCEPT
-A OUTPUT -d 127.0.0.11 -j REJECT
-A OUTPUT -o lo -j ACCEPT
-A OUTPUT -p tcp --sport 9050 -m conntrack --ctstate ESTABLISHED -j ACCEPT
-A OUTPUT -o wg0 -j ACCEPT
-A OUTPUT -p udp -d {endpoint} --dport {port} -j ACCEPT
COMMIT
"""
    run("iptables-restore", data=rules)
    run("ip6tables-restore", data="*filter\n:INPUT DROP [0:0]\n:FORWARD DROP [0:0]\n:OUTPUT DROP [0:0]\nCOMMIT\n")


def health():
    rows = run("wg", "show", "wg0", "latest-handshakes").strip().splitlines()
    return len(rows) == 1 and 0 <= time.time() - int(rows[0].split()[1]) < 180


def main():
    if not Path("/.dockerenv").exists() or os.environ.get("PHANTOM_CONTAINER") != "1":
        raise RuntimeError("Run only inside the dedicated Phantom gateway container")
    if len(sys.argv) == 2 and sys.argv[1] == "health":
        return 0 if health() else 1
    p = profile(Path("/run/secrets/wireguard.conf").read_text())
    resolvers = [line.split()[1] for line in Path("/etc/resolv.conf").read_text().splitlines() if line.strip().startswith("nameserver ")]
    if resolvers != [p["dns"]]:
        raise RuntimeError("Mount the profile DNS resolver file before starting the gateway")
    routes = json.loads(run("ip", "-j", "-4", "route", "show", "default"))
    if len(routes) != 1 or not routes[0].get("gateway"):
        raise RuntimeError("Exactly one underlay default route is required")
    route = routes[0]
    firewall(p["endpoint"], p["port"])
    run("ip", "route", "replace", p["endpoint"] + "/32", "via", route["gateway"], "dev", route["dev"])
    run("ip", "link", "add", "wg0", "type", "wireguard")
    run("wg", "setconf", "wg0", "/dev/stdin", data=p["config"])
    run("ip", "address", "add", p["address"], "dev", "wg0")
    run("ip", "link", "set", "dev", "wg0", "mtu", str(p["mtu"]), "up")
    run("ip", "route", "replace", "default", "dev", "wg0")
    # Also deny Docker's embedded DNS (DNAT occurs before filter OUTPUT).
    # It must not create a bypass around the VPN resolver.
    print("WireGuard interface ready; waiting for authenticated peer; IPv6 blocked", flush=True)
    stopping = False
    def stop(*_):
        nonlocal stopping
        stopping = True
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    try:
        while not stopping:
            time.sleep(1)
    finally:
        run("ip", "link", "del", "wg0")
        # Keep DROP policies: stopping must never restore direct egress.
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except RuntimeError as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(1)
    except OSError as exc:
        print(f"Gateway file/system error: errno={exc.errno} path={exc.filename}", file=sys.stderr)
        sys.exit(1)
    except (ValueError, KeyError, configparser.Error, subprocess.SubprocessError):
        print("WireGuard unavailable; check the profile and gateway capabilities. No direct fallback.", file=sys.stderr)
        sys.exit(1)
