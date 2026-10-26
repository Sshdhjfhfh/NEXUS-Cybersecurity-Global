#!/usr/bin/env python3
"""Instala las habilidades incluidas sin sobrescribir archivos del usuario."""
import argparse
import os
from pathlib import Path


def seed(source: Path, destination: Path) -> int:
    if source.is_symlink() or not source.is_dir():
        raise ValueError("invalid bundled skills directory")
    destination.mkdir(parents=True, exist_ok=True, mode=0o700)
    if destination.is_symlink():
        raise ValueError("skills destination cannot be a symlink")
    count = 0
    for package in sorted(source.iterdir()):
        if package.is_symlink() or not package.is_dir():
            continue
        target = destination / package.name
        # A skill is a versioned unit: never mix new defaults into a user copy.
        if target.exists() or target.is_symlink():
            continue
        import shutil
        import tempfile
        with tempfile.TemporaryDirectory(prefix=".seed-", dir=destination) as stage:
            staged = Path(stage) / package.name
            if any(p.is_symlink() for p in package.rglob("*")):
                raise ValueError("bundled skill contains a symlink")
            shutil.copytree(package, staged)
            try:
                os.rename(staged, target)
            except FileExistsError:
                continue
        count += 1
    return count


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("source", type=Path)
    parser.add_argument("destination", type=Path)
    args = parser.parse_args()
    print(f"[skills] seeded {seed(args.source, args.destination)} bundled packages")
