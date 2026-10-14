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
