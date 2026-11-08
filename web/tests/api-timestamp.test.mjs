import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import ts from 'typescript';

// Compile the real API module without starting Vite or contacting a bank.
// Replace only the Vite environment expression, not the validation logic.
const source = readFileSync(new URL('../src/api.ts', import.meta.url), 'utf8')
  .replace("import.meta.env.VITE_API_BASE_URL", "undefined");
const compiled = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
  reportDiagnostics: true
});
assert.equal(compiled.diagnostics?.length ?? 0, 0);
const { validPostedAt } = await import(
  'data:text/javascript;base64,' + Buffer.from(compiled.outputText).toString('base64')
);

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
