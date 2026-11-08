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

const methodStart = source.indexOf('async allTransactions(id:string):Promise<Transaction[]>');
const methodEnd = source.indexOf('\n }\n};', methodStart);
assert.ok(methodStart >= 0 && methodEnd > methodStart, 'Expected allTransactions method');
const methodSource = source.slice(methodStart, methodEnd + 3)
  .replace('async allTransactions(id:string):Promise<Transaction[]>', 'async function allTransactions(id)')
  .replace(/const result:Transaction\[\]=\[\]/, 'const result=[]')
  .replace(/const seen=new Set<string>\(\)/, 'const seen=new Set()')
  .replace(/request<\{transactions:Transaction\[\];limit:number;offset:number;has_more\?:boolean\}>/g, 'request')
  .replace(/request<\{transactions:Transaction\[\];limit:number;offset:number\}>/g, 'request');
function makeLoader(request) {
  return new Function('request', 'connectionPath', 'validPostedAt',
    methodSource + '\nreturn allTransactions;')(request, id => '/v1/connections/' + encodeURIComponent(id), validPostedAt);
}
const transaction = (id) => ({
  provider_external_id: String(id), amount_cents: -575, currency: 'USD',
  description: 'Corner Coffee', posted_at: '2026-11-08T12:00:00Z'
});
const page = (offset, records, has_more) => ({offset,limit:200,transactions:records,has_more});

test('loads all pages without dropping transactions or changing order', async () => {
  const records = Array.from({length: 201}, (_, i) => transaction(i));
  const calls=[];
  const load = makeLoader(async path => {
    const offset=Number(new URL(path,'http://localhost').searchParams.get('offset'));
    calls.push(offset);
    return page(offset, records.slice(offset,offset+200), offset===0);
  });
  assert.deepEqual(await load('mock-bank'), records);
  assert.deepEqual(calls,[0,200,0]);
});

test('rejects duplicate IDs across pages instead of double counting', async () => {
  const first=Array.from({length:200},(_,i)=>transaction(i));
  const load=makeLoader(async path => {
    const offset=Number(new URL(path,'http://localhost').searchParams.get('offset'));
    return page(offset,offset===0?first:[transaction(0)],offset===0);
  });
  await assert.rejects(load('mock-bank'),/Duplicate transaction ID/);
});

test('rejects changing history during pagination', async () => {
  const first=Array.from({length:200},(_,i)=>transaction(i));
  let headReads=0;
  const load=makeLoader(async path => {
    const offset=Number(new URL(path,'http://localhost').searchParams.get('offset'));
    if(offset===200)return page(200,[transaction(200)],false);
    headReads++;
    return page(0,headReads===1?first:[transaction('changed'),...first.slice(1)],true);
  });
  await assert.rejects(load('mock-bank'),/history changed during retrieval/);
});

test('rejects malformed records before displaying spending', async () => {
  const load=makeLoader(async () => page(0,[{...transaction(1),posted_at:'2026-02-30T12:00:00Z'}],false));
  await assert.rejects(load('mock-bank'),/Invalid transaction timestamp/);
});


test('handles exactly one full transaction page followed by an empty page', async () => {
  const records=Array.from({length:200},(_,i)=>transaction(i));
  const calls=[];
  const load=makeLoader(async path => {
    const offset=Number(new URL(path,'http://localhost').searchParams.get('offset'));
    calls.push(offset);
    return page(offset,offset===0?records:[],offset===0);
  });
  assert.deepEqual(await load('mock-bank'),records);
  assert.deepEqual(calls,[0,200,0]);
});

test('rejects inconsistent pagination metadata before computing analytics', async () => {
  const load=makeLoader(async () => page(200,[transaction(1)],false));
  await assert.rejects(load('mock-bank'),/page metadata does not match/);
});

test('rejects invalid financial amounts rather than silently summing them', async () => {
  for(const amount of [Number.NaN,Number.POSITIVE_INFINITY,1.25,Number.MAX_SAFE_INTEGER+1]){
    const load=makeLoader(async () => page(0,[{...transaction(1),amount_cents:amount}],false));
    await assert.rejects(load('mock-bank'),/Malformed transaction record/);
  }
});

test('rejects incomplete pages that claim more records exist', async () => {
  const load=makeLoader(async () => page(0,[transaction(1)],true));
  await assert.rejects(load('mock-bank'),/incomplete despite a continuation hint/);
});
