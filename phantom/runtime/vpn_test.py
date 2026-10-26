"""Real kernel WireGuard, HTTP, DNS and kill-switch test in disposable containers."""
import base64
import json
from pathlib import Path
import secrets
import subprocess
import sys
import tempfile
import time
import uuid

HERE = Path(__file__).resolve().parent
IMAGE = "nexus-phantom:local"


def docker(*args, data=None, check=True, timeout=30):
    result = subprocess.run(["docker", *args], input=data, capture_output=True, text=True, timeout=timeout)
    if check and result.returncode:
        raise RuntimeError("Docker probe failed: " + result.stderr[-1200:])
    return result


SERVER = r'''
import json, subprocess, threading, socket, struct
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
p=json.loads(Path('/run/secrets/server.json').read_text())
def run(*args,data=None):
 subprocess.run(args,input=data,text=True,check=True,capture_output=True)
run('ip','link','add','wg0','type','wireguard')
run('wg','setconf','wg0','/dev/stdin',data='[Interface]\nPrivateKey = '+p['private']+'\nListenPort = 51820\n[Peer]\nPublicKey = '+p['peer']+'\nAllowedIPs = 10.73.0.2/32\n')
run('ip','addr','add','10.73.0.1/30','dev','wg0')
run('ip','link','set','wg0','up')
class Handler(BaseHTTPRequestHandler):
 def do_GET(self):
  body=json.dumps({'peer':self.client_address[0],'payload':'wireguard-real-transport'}).encode()
  self.send_response(200);self.send_header('Content-Length',str(len(body)));self.end_headers();self.wfile.write(body)
 def log_message(self,*args): pass
def dns():
 s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.bind(('10.73.0.1',53))
 while True:
  packet,addr=s.recvfrom(512)
  # Authoritative response for this isolated test zone, never forwards queries.
  end=12
  while end<len(packet) and packet[end]: end+=packet[end]+1
  end+=5
  if end>len(packet): continue
  question=packet[12:end];qtype=struct.unpack('!H',packet[end-4:end-2])[0]
  answer=b'\xc0\x0c'+struct.pack('!HHIH',1,1,0,4)+socket.inet_aton('10.73.0.1') if qtype==1 else b''
  s.sendto(packet[:2]+struct.pack('!HHHHH',0x8180,1,bool(answer),0,0)+question+answer,addr)
threading.Thread(target=dns,daemon=True).start()
HTTPServer(('0.0.0.0',8080),Handler).serve_forever()
'''


def main():
    tag = "nexus-wg-test-" + uuid.uuid4().hex[:10]
    server, client = tag + "-server", tag + "-client"
    report = {"tested_at": time.time(), "transport": "Linux kernel WireGuard", "checks": {}}
    root = HERE / "artifacts"
    root.mkdir(exist_ok=True)
    try:
        with tempfile.TemporaryDirectory(prefix="wg-", dir=root) as temp:
            temp = Path(temp)
            private = [base64.b64encode(secrets.token_bytes(32)).decode() for _ in range(2)]
            public = [docker("run", "--rm", "-i", "--network", "none", IMAGE, "wg", "pubkey", data=k+"\n").stdout.strip() for k in private]
            server_file = temp / "server.json"
            server_file.write_text(json.dumps({"private": private[0], "peer": public[1]}))
            server_file.chmod(0o600)
            docker("network", "create", tag)
            common = ["--network", tag, "--user", "0:0", "--cap-drop", "ALL", "--cap-add", "NET_ADMIN", "--cap-add", "DAC_READ_SEARCH", "--security-opt", "no-new-privileges:true", "--read-only", "--tmpfs", "/run:size=4m,noexec,nosuid"]
            docker("run", "-d", "--name", server, *common, "--mount", f"type=bind,src={server_file},dst=/run/secrets/server.json,readonly", IMAGE, "python3", "-c", SERVER)
            address = docker("inspect", "--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", server).stdout.strip()
            # Prove the underlay destination exists before the negative test.
            ready = False
            for _ in range(20):
                result = docker("exec", server, "curl", "-fsS", "--max-time", "1", f"http://{address}:8080", check=False)
                if result.returncode == 0:
                    ready = True
                    break
                time.sleep(0.2)
            if not ready:
                raise RuntimeError("WireGuard test peer did not start")
            profile = temp / "client.conf"
            profile.write_text(f"[Interface]\nPrivateKey = {private[1]}\nAddress = 10.73.0.2/30\nDNS = 10.73.0.1\n[Peer]\nPublicKey = {public[0]}\nEndpoint = {address}:51820\nAllowedIPs = 0.0.0.0/0\nPersistentKeepalive = 1\n")
            profile.chmod(0o600)
            resolver = temp / "resolv.conf"
            resolver.write_text("nameserver 10.73.0.1\noptions timeout:1 attempts:1\n")
            docker("run", "-d", "--name", client, *common, "-e", "PHANTOM_CONTAINER=1", "--mount", f"type=bind,src={resolver},dst=/etc/resolv.conf,readonly", "--mount", f"type=bind,src={profile},dst=/run/secrets/wireguard.conf,readonly", IMAGE, "python3", "/opt/phantom/wireguard.py")
            for _ in range(20):
                result = docker("exec", client, "curl", "-fsS", "--max-time", "2", "http://10.73.0.1:8080", check=False)
                if result.returncode == 0:
                    break
                time.sleep(0.3)
            if result.returncode != 0:
                logs = docker("logs", client, check=False)
                raise RuntimeError("No traffic crossed the tunnel: " + (logs.stdout + logs.stderr)[-1000:])
            response = json.loads(result.stdout)
            report["checks"]["http_crossed_tunnel"] = response == {"peer": "10.73.0.2", "payload": "wireguard-real-transport"}
            report["checks"]["authenticated_handshake"] = docker("exec", client, "python3", "/opt/phantom/wireguard.py", "health", check=False).returncode == 0
            transfer = docker("exec", client, "wg", "show", "wg0", "transfer").stdout.split()
            report["checks"]["encrypted_bytes_both_directions"] = len(transfer) == 3 and int(transfer[1]) > 0 and int(transfer[2]) > 0
            dns = docker("exec", client, "curl", "-fsS", "--max-time", "5", "http://vpn-proof.invalid:8080", check=False)
            report["checks"]["dns_over_tunnel"] = dns.returncode == 0 and json.loads(dns.stdout).get("peer") == "10.73.0.2"
            # Establishment does not permit bypassing the tunnel to a LAN/underlay IP.
            direct_args = ["exec", client, "curl", "-fsS", "--max-time", "2", f"http://{address}:8080"]
            report["checks"]["underlay_tcp_blocked_while_up"] = docker(*direct_args, check=False).returncode != 0
            docker("exec", client, "ip", "link", "del", "wg0")
            report["checks"]["interface_removed_blocks_direct"] = docker(*direct_args, check=False).returncode != 0
            blocked = docker("exec", client, "curl", "-fsS", "--max-time", "3", "http://vpn-proof.invalid:8080", check=False)
            report["checks"]["interface_removed_blocks_dns"] = blocked.returncode != 0
            report["checks"]["ipv6_drop_policy"] = "-P OUTPUT DROP" in docker("exec", client, "ip6tables", "-S", "OUTPUT").stdout
            report["passed"] = all(report["checks"].values())
            report["limits"] = "Paquetes y claves reales entre dos contenedores. No acredita un proveedor remoto ni una salida VPN a Internet; requiere un perfil del usuario."
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as exc:
        report["passed"] = False
        report["error"] = str(exc)
    finally:
        for container in (client, server):
            docker("rm", "-f", container, check=False)
        docker("network", "rm", tag, check=False)
    (root / "vpn-check.json").write_text(json.dumps(report, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
