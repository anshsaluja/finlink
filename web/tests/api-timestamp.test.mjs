import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import * as ts from 'typescript';

// Compile the real exported validator in isolation. Loading the entire API module
// would also evaluate browser-only import.meta.env and unrelated request helpers.
const source = readFileSync(new URL('../src/api.ts', import.meta.url), 'utf8');
const file = ts.createSourceFile('api.ts', source, ts.ScriptTarget.ES2022, true, ts.ScriptKind.TS);
const validator = file.statements.find(node =>
  ts.isFunctionDeclaration(node) && node.name?.text === 'validPostedAt' &&
  node.modifiers?.some(modifier => modifier.kind === ts.SyntaxKind.ExportKeyword)
);
assert.ok(validator, 'Expected the exported validPostedAt implementation in src/api.ts');
const compiled = ts.transpileModule(validator.getText(file), {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS },
  reportDiagnostics: true
});
assert.equal(compiled.diagnostics?.length ?? 0, 0);
// Evaluate the isolated validator as CommonJS to avoid data-URL ESM loader
// differences between local Node and GitHub Actions environments.
const moduleExports = {};
new Function('exports', compiled.outputText)(moduleExports);
const { validPostedAt } = moduleExports;
assert.equal(typeof validPostedAt, 'function', 'Compiled validator must be exported');

test('accepts valid UTC, offset and leap-day transaction timestamps', () => {
  for (const value of [
    '2024-02-29T23:59:59Z',
    '2026-11-08T01:30:00-05:00',
    '2026-01-01T00:00:00.123456789+14:00',
    '2000-02-29T00:00:00Z',
    '0096-02-29T00:00:00Z'
  ]) assert.equal(validPostedAt(value), true, value);
});

test('rejects impossible dates, ambiguous times and malformed offsets', () => {
  for (const value of [
    '2026-02-29T12:00:00Z',
    '1900-02-29T12:00:00Z',
    '2026-04-31T12:00:00Z',
    '2026-13-01T00:00:00Z',
    '2026-01-01T24:00:00Z',
    '2026-01-01T00:60:00Z',
    '2026-01-01T00:00:60Z',
    '2026-01-01T00:00:00',
    '2026-01-01',
    '2026-01-01T00:00:00+14:30',
    '2026-01-01T00:00:00+15:00',
    '2026-01-01T00:00:00-01:60',
    '2026-01-01T00:00:00.1234567890Z',
    'not-a-date'
  ]) assert.equal(validPostedAt(value), false, value);
});
