#!/usr/bin/env bash
set -euo pipefail
ROOT="$HOME/nexus"
mkdir -p "$ROOT"
cd "$ROOT"
GREEN='\033[0;32m'; BLUE='\033[0;34m'; NC='\033[0m'
log() { echo -e "${BLUE}▶${NC} $1"; }
ok()  { echo -e "${GREEN}✅${NC} $1"; }

echo "═══════════════════════════════════════"
echo "  NEXUS IA v6.0 — INSTALADOR"
echo "═══════════════════════════════════════"

log "Creando estructura..."

mkdir -p i18n/locales
mkdir -p llm_gateway/tui llm_gateway/web llm_gateway/failover llm_gateway/providers
mkdir -p web/app web/components web/public
mkdir -p proto/nexus/v6/common
mkdir -p gen/go gen/python gen/rust gen/ts
mkdir -p phantom/vpn phantom/onion phantom/mixnet phantom/cover_traffic
mkdir -p phantom/proxy phantom/obfs phantom/garlic phantom/stego
mkdir -p phantom/timing phantom/identity
mkdir -p agents/red agents/blue agents/prediction
mkdir -p swarm watchdog
mkdir -p lab/dvwa lab/metasploitable lab/vulnhub
mkdir -p lab/active_directory lab/network
mkdir -p evolve memory knowledge_bus
mkdir -p digital_twin xai federated ethical_boundary
mkdir -p graph_rag vckg zero_day symbolic_ai ai_scientist
mkdir -p cti_feed llmjacking_guard oracle_defense nexus_bench
mkdir -p research evoexploit selfimprove tom
mkdir -p nhi_manager abm ctf_engine zero_trust
mkdir -p rsi_detector ans adversarial_rt governance
mkdir -p orchestrator session crypto observability
mkdir -p policy secrets config scripts
mkdir -p infra/k8s infra/helm infra/pki infra/terraform
mkdir -p docs papers
mkdir -p tests/integration tests/chaos tests/i18n tests/security

ok "Estructura: $(find . -type d | wc -l) directorios"

log "Creando hardware.yaml..."
cat > config/hardware.yaml << 'YAML_EOF'
tiers:
  nano:
    min_vram_gb: 0
    min_ram_gb: 4
    models:
      agents: qwen3:1.7b
      ethical_boundary: qwen3:1.7b
      reasoning: deepseek-r1:1.5b
    parallelism: { swarm_agents: 2, ctf_parallel: 1 }
  micro:
    min_vram_gb: 2
    min_ram_gb: 8
    models:
      agents: qwen3:4b
      ethical_boundary: qwen3:4b
      reasoning: deepseek-r1:1.5b
    parallelism: { swarm_agents: 4, ctf_parallel: 2 }
  standard:
    min_vram_gb: 4
    min_ram_gb: 12
    models:
      agents: qwen3:8b
      ethical_boundary: qwen3:8b
      reasoning: deepseek-r1:7b
    parallelism: { swarm_agents: 8, ctf_parallel: 4 }
  pro:
    min_vram_gb: 8
    min_ram_gb: 16
    models:
      agents: qwen3:14b
      ethical_boundary: qwen3:8b
      reasoning: deepseek-r1:14b
    parallelism: { swarm_agents: 8, ctf_parallel: 6 }
  ultra:
    min_vram_gb: 16
    min_ram_gb: 32
    models:
      agents: qwen3:32b
      ethical_boundary: qwen3:8b
      reasoning: deepseek-r1:32b
    parallelism: { swarm_agents: 16, ctf_parallel: 8 }
YAML_EOF
ok "config/hardware.yaml"

log "Creando scripts de detección..."
cat > scripts/hardware_probe.py << 'PY_EOF'
#!/usr/bin/env python3
import json, subprocess, platform, os
from pathlib import Path

def run(cmd):
    try:
        return subprocess.check_output(cmd, shell=True, text=True,
                                       stderr=subprocess.DEVNULL).strip()
    except Exception:
        return ""

def main():
    name = ""
    if platform.system() == "Linux":
        with open("/proc/cpuinfo") as f:
            for line in f:
                if "model name" in line:
                    name = line.split(":")[1].strip()
                    break
    ram = 0
    if platform.system() == "Linux":
        with open("/proc/meminfo") as f:
            for line in f:
                if line.startswith("MemTotal"):
                    ram = round(int(line.split()[1]) / 1024 / 1024, 2)
                    break
    gpus = []
    smi = run("nvidia-smi --query-gpu=name,memory.total --format=csv,noheader,nounits")
    if smi:
        for line in smi.splitlines():
            parts = [p.strip() for p in line.split(",")]
            if len(parts) == 2:
                gpus.append({"name": parts[0], "vram_mb": int(parts[1])})
    profile = {
        "cpu": {"name": name, "logical_cores": os.cpu_count() or 0},
        "ram_gb": ram,
        "gpus": gpus,
        "wsl": "microsoft" in platform.uname().release.lower(),
    }
    out = Path.home() / "nexus" / "config" / "hardware_profile.json"
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(profile, indent=2))
    print(json.dumps(profile, indent=2))

if __name__ == "__main__":
    main()
PY_EOF
chmod +x scripts/hardware_probe.py

cat > scripts/tier_selector.py << 'PY_EOF'
#!/usr/bin/env python3
import json, yaml
from pathlib import Path

home = Path.home() / "nexus"
profile = json.loads((home / "config" / "hardware_profile.json").read_text())
tiers = yaml.safe_load((home / "config" / "hardware.yaml").read_text())["tiers"]

vram = max((g["vram_mb"] for g in profile["gpus"]), default=0) / 1024
ram = profile["ram_gb"]

chosen = "nano"
for name in ["ultra", "pro", "standard", "micro", "nano"]:
    t = tiers[name]
    if vram >= t["min_vram_gb"] and ram >= t["min_ram_gb"]:
        chosen = name
        break

out = {"tier": chosen, "hardware": profile, "config": tiers[chosen]}
(home / "config" / "active_tier.yaml").write_text(yaml.safe_dump(out, sort_keys=False))
print(f"Tier: {chosen} | VRAM: {vram:.1f}GB | RAM: {ram:.1f}GB")
PY_EOF
chmod +x scripts/tier_selector.py

cat > scripts/generate_compose.py << 'PY_EOF'
#!/usr/bin/env python3
print("generate_compose.py placeholder — se implementará en FASE 1")
PY_EOF
chmod +x scripts/generate_compose.py
ok "scripts listos"

log "Creando Makefile..."
cat > Makefile << 'MK_EOF'
.PHONY: help detect tier up down logs ps clean

help:
	@echo "Nexus IA v6.0 — Comandos:"
	@echo "  make detect  - Detecta hardware"
	@echo "  make tier    - Selecciona tier"
	@echo "  make up      - Arranca Nexus"
	@echo "  make down    - Detiene Nexus"
	@echo "  make logs    - Ver logs"

detect:
	@python3 scripts/hardware_probe.py

tier: detect
	@python3 scripts/tier_selector.py
	@echo "---"
	@cat config/active_tier.yaml

up:
	@docker compose up -d

down:
	@docker compose down

logs:
	@docker compose logs -f

ps:
	@docker ps

clean:
	@docker compose down -v
MK_EOF
ok "Makefile"

log "Creando .env..."
cat > .env << 'ENV_EOF'
NEXUS_DEFAULT_LANG=es
OLLAMA_HOST=http://host.docker.internal:11434
MODEL_AGENTS=qwen3:4b
MODEL_ETHICAL=qwen3:4b
MODEL_REASONING=deepseek-r1:1.5b
ENV_STATE=development
ENV_EOF
ok ".env"
cp .env .env.example

log "Creando .gitignore..."
cat > .gitignore << 'GI_EOF'
__pycache__/
*.py[cod]
.venv/
node_modules/
.next/
.env
config/hardware_profile.json
config/active_tier.yaml
secrets/*.pem
secrets/*.key
secrets/*.crt
gen/
*.log
logs/
.DS_Store
GI_EOF
ok ".gitignore"

log "Detectando hardware..."
python3 scripts/hardware_probe.py > /dev/null
python3 scripts/tier_selector.py

echo ""
echo "═══════════════════════════════════════"
ok "INSTALACIÓN COMPLETA"
echo "═══════════════════════════════════════"
echo "Carpetas: $(find . -type d | wc -l)"
echo "Archivos: $(find . -type f | wc -l)"
echo ""
echo "Siguiente: make tier && cat config/active_tier.yaml"
