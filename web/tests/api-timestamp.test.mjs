import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
// Extract the real validator without loading browser-only import.meta.env.
// The validator has no nested function declarations or external dependencies.
const source = readFileSync(new URL('../src/api.ts', import.meta.url), 'utf8');
const start = source.indexOf('export function validPostedAt(');
const end = source.indexOf('\nconst connectionPath=', start);
assert.ok(start >= 0 && end > start, 'Expected validPostedAt before connectionPath');
const extracted = source.slice(start, end).trim();
assert.match(extracted, /^export function validPostedAt\(value:string\):boolean \{/);
const js = extracted.replace('export function validPostedAt(value:string):boolean', 'function validPostedAt(value)');
const validPostedAt = new Function(js + '\nreturn validPostedAt;')();

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
