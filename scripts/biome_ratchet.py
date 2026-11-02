#!/usr/bin/env python3
"""Trinquete de errores de Biome para la web.

Compara los errores de `biome check` (lint, formato e imports) por archivo con una línea base versionada
(`web/biome-baseline.json`). Falla si un archivo supera su línea base o si un
archivo sin línea base tiene errores. Así la deuda existente no bloquea la CI,
pero no puede crecer. Cuando un archivo mejora, se avisa para bajar su línea
base con `--update`, que nunca la sube para un archivo ya presente.

Uso:
  python3 scripts/biome_ratchet.py                 # ejecuta biome en web/
  python3 scripts/biome_ratchet.py --report r.json # usa un informe ya generado
  python3 scripts/biome_ratchet.py --update        # reescribe la línea base
"""

import argparse
import collections
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
WEB = ROOT / "web"
BASELINE = WEB / "biome-baseline.json"


def run_biome() -> dict:
    proc = subprocess.run(
        [str(WEB / "node_modules/.bin/biome"), "check", "--reporter=json", "--max-diagnostics=none", "."],
        cwd=WEB,
        capture_output=True,
        text=True,
    )
    if not proc.stdout.strip():
        sys.exit(f"biome no produjo informe (rc={proc.returncode}): {proc.stderr.strip()[:500]}")
    return json.loads(proc.stdout)


def error_counts(report: dict) -> dict:
    counts = collections.Counter()
    for diag in report.get("diagnostics", []):
        if diag.get("severity") != "error":
            continue
        path = diag.get("location", {}).get("path")
        if isinstance(path, dict):
            path = path.get("file")
        counts[path or "<desconocido>"] += 1
    return dict(counts)


def compare(current: dict, baseline: dict) -> tuple:
    """Devuelve (regresiones, mejoras) como listas de (archivo, actual, base)."""
    worse, better = [], []
    for path in sorted(set(current) | set(baseline)):
        now, base = current.get(path, 0), baseline.get(path, 0)
        if now > base:
            worse.append((path, now, base))
        elif now < base:
            better.append((path, now, base))
    return worse, better


def updated_baseline(current: dict, baseline: dict) -> dict:
    """Nunca sube la base de un archivo ya presente; los nuevos entran con su cuenta."""
    out = {}
    for path, now in current.items():
        out[path] = min(now, baseline[path]) if path in baseline else now
    return dict(sorted(out.items()))


def main(argv=None) -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--report", help="informe JSON de biome ya generado")
    ap.add_argument("--baseline", default=str(BASELINE))
    ap.add_argument("--update", action="store_true", help="reescribe la línea base")
    args = ap.parse_args(argv)

    report = json.loads(Path(args.report).read_text(encoding="utf-8")) if args.report else run_biome()
    current = error_counts(report)
    base_path = Path(args.baseline)
    baseline = json.loads(base_path.read_text(encoding="utf-8")) if base_path.exists() else {}

    if args.update:
        new = updated_baseline(current, baseline)
        base_path.write_text(json.dumps(new, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
        print(f"línea base: {sum(new.values())} errores en {len(new)} archivos")
        return 0

    worse, better = compare(current, baseline)
    print(f"biome: {sum(current.values())} errores (línea base {sum(baseline.values())})")
    for path, now, base in better:
        print(f"  mejora  {path}: {base} -> {now} (baja la base con --update)")
    for path, now, base in worse:
        print(f"  REGRESIÓN {path}: {base} -> {now}")
    return 1 if worse else 0


if __name__ == "__main__":
    sys.exit(main())
