#!/usr/bin/env python3
"""Extract, apply and audit Spanish frontend translations without regex code edits.

Run from the repository root. Requires Node and web's TypeScript dependency.
--dry-run performs no writes, including locale synchronization.
"""
import argparse
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parent.parent


def plan(source, translations):
    files = sorted(str(p) for p in source.rglob('*') if p.suffix in ('.ts', '.tsx'))
    result = subprocess.run(
        ['node', str(ROOT / 'scripts/frontend_i18n.cjs')],
        input=json.dumps({'files': files, 'translations': translations}),
        text=True, capture_output=True, check=True, cwd=ROOT,
    )
    return json.loads(result.stdout)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--dry-run', action='store_true')
    parser.add_argument('--extract', type=Path, help='Write extracted Chinese literals as JSON; use - for stdout')
    parser.add_argument('--source-dir', type=Path, default=ROOT / 'web/src')
    args = parser.parse_args()
    catalog_path = ROOT / 'i18n/locales/es.json'
    catalog = json.loads(catalog_path.read_text())
    translations = {k: v for k, v in catalog.items() if isinstance(v, str)}
    result = plan(args.source_dir.resolve(), translations)
    extracted = [{'file': f['file'], **entry} for f in result['files'] for entry in f.get('extracted', [])]
    if args.extract and not args.dry_run:
        payload = json.dumps(extracted, ensure_ascii=False, indent=2) + '\n'
        if str(args.extract) == '-':
            print(payload, end='')
        else:
            args.extract.write_text(payload)
    if args.extract:
        print(f'Extracted {len(extracted)} Chinese literals')
        return
    changes = [f for f in result['files'] if 'content' in f]
    print(f"{'DRY RUN' if args.dry_run else 'APPLY'}: {len(changes)} files, "
          f"{sum(f['changes'] for f in changes)} replacements, {len(extracted)} Chinese literals")
    if args.dry_run:
        return
    for entry in changes:
        Path(entry['file']).write_text(entry['content'])
    catalog.setdefault('ui', {}).update(result['ui'])
    serialized = json.dumps(catalog, ensure_ascii=False, indent=2) + '\n'
    catalog_path.write_text(serialized)
    (ROOT / 'web/public/locales/es.json').write_text(serialized)


if __name__ == '__main__':
    main()
