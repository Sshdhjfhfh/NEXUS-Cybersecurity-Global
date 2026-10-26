"""Reproducible loopback-only audit of current Phantom behavior."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parent.parent


def audit():
    command = ["go", "test", "-race", "-count=1", "-json", "./phantom/...", "./terminal/checks"]
    completed = subprocess.run(command, cwd=ROOT, capture_output=True, text=True, timeout=300, check=False)
    passed, failed = [], []
    for line in completed.stdout.splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if event.get("Test") and event.get("Action") in {"pass", "fail"}:
            (passed if event["Action"] == "pass" else failed).append(event["Test"])
    production_imports = []
    for tree in ("cmd", "orchestrator", "agents", "core", "lab", "llm_gateway"):
        for path in (ROOT / tree).rglob("*.go"):
            if not path.name.endswith("_test.go") and '"github.com/nexus-ia/nexus/phantom/' in path.read_text():
                production_imports.append(str(path.relative_to(ROOT)))
    fingerprint = hashlib.sha256()
    for path in sorted((ROOT / "phantom").rglob("*.go")):
        fingerprint.update(str(path.relative_to(ROOT)).encode())
        fingerprint.update(path.read_bytes())
    probes = {
        "proxy_rechaza_transporte_ausente": "TestDefaultProxyRejectsMissingTransport" in passed,
        "onion_cifra_sin_conectar_a_relays": "TestOnionSendBuildsCiphertextWithoutContactingRelays" in passed,
        "sin_fallback_si_transporte_configurado_falla": "TestConfiguredDialerFailureDoesNotFallBackToDirect" in passed,
    }
    report = {"tested_at": time.time(), "source_sha256": fingerprint.hexdigest(),
              "suite_ok": completed.returncode == 0, "tests_passed": len(passed), "tests_failed": failed,
              "probes": probes, "production_imports": production_imports,
              "anonymous_network_verified": False,
              "conclusion": "El proxy exige transporte; la red Tor/WireGuard se valida aparte con nexus phantom check y vpn-test. Onion/Mixnet propios siguen siendo experimentales.",
              "evidence": [
                  "Onion Send devuelve bytes cifrados y contadores sin contactar ninguno de los tres relays locales.",
                  "El proxy rechaza el arranque si falta un transporte explícito.",
                  "Un Dialer explícito que rechaza sí impide la conexión directa en esa configuración.",
                  "Mixnet construye paquetes y usa callbacks; no aporta por sí mismo un transporte entre nodos.",
              ], "limits": "Pruebas de componentes y transporte local; no certifican anonimato ni una salida IP protegida en Internet."}
    if production_imports:
        report["evidence"].append("Existen imports de Phantom en producción; revisar manualmente su composición.")
    else:
        report["evidence"].append("No hay imports de Phantom en el código de producción inspeccionado.")
    if completed.returncode:
        report["diagnostic"] = (completed.stderr + completed.stdout)[-12000:]
        report["conclusion"] = "La suite falló; no se debe dar por validado Phantom. Consulta el diagnóstico."
    output = ROOT / "terminal" / "artifacts"
    output.mkdir(parents=True, exist_ok=True)
    (output / "phantom-audit.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return completed.returncode


if __name__ == "__main__":
    try:
        sys.exit(audit())
    except (OSError, subprocess.TimeoutExpired) as exc:
        print("No se pudo completar la auditoría: " + str(exc), file=sys.stderr)
        sys.exit(1)
