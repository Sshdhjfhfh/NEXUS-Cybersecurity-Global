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
