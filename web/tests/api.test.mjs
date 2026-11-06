import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { stripTypeScriptTypes } from 'node:module';

// Compile the actual API module, not a duplicate of its validation logic.
// Replace only Vite's build-time base URL expression for Node's test runner.
const source = readFileSync(new URL('../src/api.ts', import.meta.url), 'utf8')
  .replace(/^const BASE = .*;$/m, "const BASE = 'http://localhost:8080';");
const compiled = stripTypeScriptTypes(source, { mode: 'strip' });
const { api, isValidTransactionTimestamp } = await import(
  'data:text/javascript;base64,' + Buffer.from(compiled).toString('base64')
);

test('accepts real UTC and offset timestamps, including leap day', () => {
  for (const value of [
    '2024-02-29T23:59:59Z',
    '2026-11-06T12:30:00.123Z',
    '2026-01-01T00:00:00+05:30',
    '2000-02-29T00:00:00-04:00',
  ]) assert.equal(isValidTransactionTimestamp(value), true, value);
});

test('rejects calendar rollovers, invalid timezones and malformed dates', () => {
  for (const value of [
    '2025-02-29T00:00:00Z',
    '1900-02-29T00:00:00Z',
    '2026-04-31T00:00:00Z',
    '2026-13-01T00:00:00Z',
    '2026-01-00T00:00:00Z',
    '2026-01-01T24:00:00Z',
    '2026-01-01T00:60:00Z',
    '2026-01-01T00:00:60Z',
    '2026-01-01T00:00:00+24:00',
    '2026-01-01T00:00:00+05:60',
    'not-a-date',
  ]) assert.equal(isValidTransactionTimestamp(value), false, value);
});

function withMockPages(pages, fn) {
  const previous = globalThis.fetch;
  let calls = 0;
  globalThis.fetch = async (url) => {
    const offset = Number(new URL(url).searchParams.get('offset'));
    const page = pages.get(offset);
    calls++;
    assert.ok(page, 'unexpected page offset ' + offset);
    return { ok: true, json: async () => page };
  };
  return Promise.resolve().then(() => fn(() => calls))
    .finally(() => { globalThis.fetch = previous; });
}

const record = (id, posted_at = '2026-11-06T12:30:00Z') => ({
  provider_external_id: id, amount_cents: -575,
  currency: 'USD', description: 'Mock coffee', posted_at,
});
const page = (offset, records) => ({ offset, limit: 200, transactions: records });

test('short valid transaction page terminates retrieval', async () => {
  await withMockPages(new Map([[0, page(0, [record('a')])]]), async (calls) => {
    assert.deepEqual(await api.allTransactions('mock-connection'), [record('a')]);
    assert.equal(calls(), 1);
  });
});

test('invalid calendar timestamp fails closed before spending totals', async () => {
  await withMockPages(new Map([[0, page(0, [record('a', '2025-02-29T00:00:00Z')])]]), async () => {
    await assert.rejects(api.allTransactions('mock-connection'), /Malformed transaction record/);
  });
});

test('duplicate IDs within a page fail closed', async () => {
  await withMockPages(new Map([[0, page(0, [record('a'), record('a')])]]), async () => {
    await assert.rejects(api.allTransactions('mock-connection'), /Duplicate transaction ID/);
  });
});

test('mismatched page metadata fails closed', async () => {
  await withMockPages(new Map([[0, { ...page(0, [record('a')]), offset: 200 }]]), async () => {
    await assert.rejects(api.allTransactions('mock-connection'), /page metadata/);
  });
});

test('retrieves all pages and verifies the first page before reporting totals', async () => {
  const first = Array.from({ length: 200 }, (_, i) => record('id-' + i));
  await withMockPages(new Map([
    [0, page(0, first)],
    [200, page(200, [record('last')])],
  ]), async (calls) => {
    const records = await api.allTransactions('mock-connection');
    assert.equal(records.length, 201);
    assert.equal(records.at(-1).provider_external_id, 'last');
    assert.equal(calls(), 3); // Includes the first-page consistency recheck.
  });
});

test('rejects duplicate transaction IDs across page boundaries', async () => {
  const first = Array.from({ length: 200 }, (_, i) => record('id-' + i));
  await withMockPages(new Map([
    [0, page(0, first)],
    [200, page(200, [record('id-0')])],
  ]), async () => {
    await assert.rejects(api.allTransactions('mock-connection'), /Duplicate transaction ID/);
  });
});

test('rejects transaction history changed while loading subsequent pages', async () => {
  const first = Array.from({ length: 200 }, (_, i) => record('id-' + i));
  const previous = globalThis.fetch;
  let headReads = 0;
  globalThis.fetch = async (url) => {
    const offset = Number(new URL(url).searchParams.get('offset'));
    if (offset === 200) return { ok: true, json: async () => page(200, [record('last')]) };
    assert.equal(offset, 0);
    headReads++;
    const records = headReads === 1 ? first : [record('inserted'), ...first.slice(0, -1)];
    return { ok: true, json: async () => page(0, records) };
  };
  try {
    await assert.rejects(api.allTransactions('mock-connection'), /Transaction history changed/);
    assert.equal(headReads, 2);
  } finally {
    globalThis.fetch = previous;
  }
});

test('encodes untrusted connection IDs as one URL path segment across reads', async () => {
  const previous = globalThis.fetch;
  const visited = [];
  globalThis.fetch = async (url) => {
    visited.push(new URL(url));
    return { ok: true, json: async () => ({ transactions: [], offset: 0, limit: 200, accounts: [] }) };
  };
  try {
    const id = 'mock/../other?token=secret#fragment';
    await api.connection(id);
    await api.accounts(id);
    await api.transactions(id);
    await api.allTransactions(id);
    assert.equal(visited.length, 4);
    for (const url of visited) {
      assert.equal(url.pathname, '/v1/connections/' + encodeURIComponent(id) + (url.pathname.endsWith('/accounts') ? '/accounts' : url.pathname.endsWith('/transactions') ? '/transactions' : ''));
      assert.equal(url.searchParams.has('token'), false);
      assert.equal(url.hash, '');
    }
  } finally {
    globalThis.fetch = previous;
  }
});


test('rejects a short page that incorrectly advertises more transactions', async () => {
  await withMockPages(new Map([[0, { ...page(0, [record('a')]), has_more: true }]]), async () => {
    await assert.rejects(api.allTransactions('mock-connection'), /incomplete despite a continuation hint/);
  });
});

test('accepts a short final page with an explicit false continuation hint', async () => {
  await withMockPages(new Map([[0, { ...page(0, [record('a')]), has_more: false }]]), async (calls) => {
    assert.equal((await api.allTransactions('mock-connection')).length, 1);
    assert.equal(calls(), 1);
  });
});

test('rejects a malformed continuation hint rather than trusting spending totals', async () => {
  await withMockPages(new Map([[0, { ...page(0, [record('a')]), has_more: 'false' }]]), async () => {
    await assert.rejects(api.allTransactions('mock-connection'), /Invalid transaction continuation metadata/);
  });
});
