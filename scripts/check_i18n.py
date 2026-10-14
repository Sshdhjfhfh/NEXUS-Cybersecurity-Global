#!/usr/bin/env python3
"""Check Spanish catalog coverage, frontend Han text and alias regressions."""
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parent.parent


def flatten(data, prefix=''):
    result = {}
    for key, value in data.items():
        key = f'{prefix}.{key}' if prefix else key
        if isinstance(value, dict):
            result.update(flatten(value, key))
        else:
            result[key] = value
    return result


def main():
    catalog = ROOT / 'i18n/locales/es.json'
    public = ROOT / 'web/public/locales/es.json'
    assert catalog.read_bytes() == public.read_bytes(), 'Spanish catalogs differ'
    translations = flatten(json.loads(catalog.read_text()))
    errors = []
    sources = [p for p in (ROOT / 'web/src').rglob('*') if p.suffix in ('.ts', '.tsx')]
    for path in sources:
        source = path.read_text()
        if re.search(r'[\u3400-\u9fff]', source):
            errors.append(f'{path.relative_to(ROOT)}: untranslated Han text')
        if re.search(r'\bt as tr\b|\btr\(', source):
            errors.append(f'{path.relative_to(ROOT)}: obsolete translation alias')
        for key in re.findall(r'\bt\([\x27"]([^\x27"]+)[\x27"]\)', source):
            if key not in translations:
                errors.append(f'{path.relative_to(ROOT)}: missing translation {key}')
    for key, value in translations.items():
        if not isinstance(value, str):
            errors.append(f'Invalid translation type: {key}')
        elif re.search(r'[\u3400-\u9fff]', value):
            errors.append(f'Untranslated Spanish value: {key}')
    if errors:
        raise SystemExit('\n'.join(errors))
    print(f'Spanish i18n checks passed: {len(sources)} sources, {len(translations)} catalog entries')


if __name__ == '__main__':
    main()
