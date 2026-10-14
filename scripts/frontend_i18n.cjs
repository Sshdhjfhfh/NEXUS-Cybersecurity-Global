// Parse frontend source with TypeScript; never rewrite comments, types or wire formats.
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const { createRequire } = require('node:module');
let ts;
try { ts = createRequire(path.resolve('web/package.json'))('typescript'); }
catch { ts = require('typescript'); }
function normalizeShadows(content, file) {
  const options = { noResolve: true, noLib: true, jsx: ts.JsxEmit.Preserve };
  const host = ts.createCompilerHost(options);
  const readSource = host.getSourceFile.bind(host);
  host.getSourceFile = (name, languageVersion, ...rest) => name === file
    ? ts.createSourceFile(name, content, languageVersion, true)
    : readSource(name, languageVersion, ...rest);
  const program = ts.createProgram([file], options, host);
  const sf = program.getSourceFile(file);
  const checker = program.getTypeChecker();
  const symbols = new Set();
  function collect(node) {
    if (ts.isCallExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === 't') {
      const symbol = checker.getSymbolAtLocation(node.expression);
      const decl = symbol?.declarations?.[0];
      if (decl && (ts.isParameter(decl) || ts.isVariableDeclaration(decl) ||
          (ts.isBindingElement(decl) && ts.isParameter(decl.parent.parent)))) symbols.add(symbol);
    }
    ts.forEachChild(node, collect);
  }
  collect(sf);
  const edits = [];
  let name = 'i18nItem';
  while (content.includes(name)) name += '_';
  function visit(node) {
    if (ts.isIdentifier(node) && node.text === 't' && symbols.has(checker.getSymbolAtLocation(node)) &&
        !(ts.isCallExpression(node.parent) && node.parent.expression === node)) {
      const shorthand = (ts.isBindingElement(node.parent) && !node.parent.propertyName) || ts.isShorthandPropertyAssignment(node.parent);
      edits.push({ start: node.getStart(sf), end: node.end, prefix: shorthand ? 't: ' : '' });
    }
    ts.forEachChild(node, visit);
  }
  visit(sf);
  for (const edit of edits.sort((a, b) => b.start - a.start)) {
    content = content.slice(0, edit.start) + edit.prefix + name + content.slice(edit.end);
  }
  return content;
}
const input = JSON.parse(fs.readFileSync(0, 'utf8'));
const han = /\p{Script=Han}/u;
const values = new Set(Object.values(input.translations).filter(v => typeof v === 'string' && v.trim().length >= 4));
const ui = {};
const output = [];
for (const file of input.files) {
  const source = fs.readFileSync(file, 'utf8');
  const sf = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true,
    file.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS);
  const edits = [];
  const extracted = [];
  function protectedNode(node) {
    for (let p = node; p; p = p.parent) {
      if (ts.isTypeNode(p) || ts.isImportDeclaration(p) || ts.isExportDeclaration(p)) return true;
      if (ts.isCallExpression(p) && /^console\./.test(p.expression.getText(sf))) return true;
    }
    return false;
  }
  function keyFor(text) {
    const key = crypto.createHash('sha256').update(text).digest('hex').slice(0, 16);
    ui[key] = text;
    return `ui.${key}`;
  }
  function visit(node) {
    if (protectedNode(node)) return;
    const literal = ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node);
    const jsx = ts.isJsxText(node);
    if (literal || jsx) {
      const raw = node.text;
      const text = jsx ? raw.trim().replace(/\s+/g, ' ') : raw;
      if (han.test(text)) extracted.push({text, line: sf.getLineAndCharacterOfPosition(node.getStart(sf)).line + 1});
      if (!text || (!han.test(text) && !values.has(text))) return;
      // Keys, directives, regexes and property names are not interface copy.
      if ((ts.isPropertyAssignment(node.parent) && node.parent.name === node) ||
          ts.isElementAccessExpression(node.parent) ||
          (ts.isCallExpression(node.parent) && node.parent.expression.getText(sf) === 't') ||
          ts.isExpressionStatement(node.parent)) return;
      const translated = input.translations[text] ?? text;
      if (han.test(translated)) return;
      const call = `t(${JSON.stringify(keyFor(translated))})`;
      let replacement = call;
      if (jsx) replacement = `${/^\s/.test(raw) ? ' ' : ''}{${call}}${/\s$/.test(raw) ? ' ' : ''}`;
      else if (ts.isJsxAttribute(node.parent)) replacement = `{${call}}`;
      edits.push({start: node.getStart(sf), end: node.end, replacement});
      return;
    }
    ts.forEachChild(node, visit);
  }
  if (!file.endsWith('/lib/i18n.ts') && !file.includes('/lib/mock/') &&
      !file.endsWith('/lib/chat-mentions.ts')) visit(sf);
  let result = source;
  for (const edit of edits.sort((a, b) => b.start - a.start)) {
    result = result.slice(0, edit.start) + edit.replacement + result.slice(edit.end);
  }
  if (edits.length && !/import\s*\{[^}]*\bt\b[^}]*\}\s*from\s*['"]@\/lib\/i18n['"]/.test(result)) {
    const directive = /^(?:['"]use client['"];?\s*)/.exec(result);
    const pos = directive ? directive[0].length : 0;
    result = result.slice(0, pos) + "import { t } from '@/lib/i18n';\n\n" + result.slice(pos);
  }
  if (result !== source) {
    result = normalizeShadows(result, file);
    output.push({file, content: result, changes: edits.length});
  }
  if (extracted.length) output.push({file, extracted});
}
process.stdout.write(JSON.stringify({files: output, ui}));
