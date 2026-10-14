"""Regression checks for safe JSX migration and a truly read-only dry run."""
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

from apply_translations import ROOT, plan


class TranslationTests(unittest.TestCase):
    def test_jsx_types_escaping_and_shadowed_import(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'fixture.tsx'
            path.write_text('''"use client";
// 保存 stays untouched in comments.
type Wire = "保存";
export const View = ({t}: {t: number}) => <button title="保存">保存{t}</button>;
''')
            result = plan(Path(directory), {'保存': 'Guardar'})
            content = result['files'][0]['content']
            self.assertIn('type Wire = "保存";', content)
            self.assertIn('// 保存 stays untouched', content)
            self.assertIn('title={t("ui.', content)
            self.assertIn('i18nItem', content)
            self.assertNotIn('title=t(', content)
            path.write_text(content)
            self.assertEqual(plan(Path(directory), {'保存': 'Guardar'})['files'], [])

    def test_dry_run_and_extraction_do_not_modify_source_or_catalogs(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'fixture.tsx'
            path.write_text('export const view = <span>保存</span>;')
            targets = [path, ROOT / 'i18n/locales/es.json', ROOT / 'web/public/locales/es.json']
            before = [hashlib.sha256(p.read_bytes()).hexdigest() for p in targets]
            for flags in [['--dry-run'], ['--extract', str(Path(directory) / 'extracted.json')]]:
                subprocess.run(['python3', str(ROOT / 'scripts/apply_translations.py'),
                                '--source-dir', directory, *flags], check=True, capture_output=True)
                self.assertEqual(before, [hashlib.sha256(p.read_bytes()).hexdigest() for p in targets])
            extracted = json.loads((Path(directory) / 'extracted.json').read_text())
            self.assertEqual(extracted[0]['text'], '保存')


if __name__ == '__main__':
    unittest.main()
