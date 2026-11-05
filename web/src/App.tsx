import {FormEvent, useEffect, useMemo, useRef, useState} from 'react';
import {api, Account, Connection, SyncRun, Transaction} from './api';

const money = (c:number, currency='USD') => new Intl.NumberFormat('en-US',{style:'currency',currency}).format(c/100);
const monthOf = (date:string) => date.slice(0,7);
const categoryOf = (description:string) => {
  const d=description.toLowerCase();
  if (/transfer|zelle|payment from|payment to/.test(d)) return 'Transfers';
  if (/refund|reversal/.test(d)) return 'Refunds';
  if (/rent|electric|utility|internet|phone bill/.test(d)) return 'Bills';
  if (/spotify|netflix|subscription/.test(d)) return 'Subscriptions';
  if (/grocery|whole foods|coffee|cafe|restaurant|starbucks|doordash|uber eats/.test(d)) return 'Food';
  if (/uber|lyft|metro|transit|gas station/.test(d)) return 'Transport';
  if (/amazon|target|walmart|shopping|store/.test(d)) return 'Shopping';
  return 'Other';
};
const merchantOf=(description:string)=>description.trim().replace(/\s+/g,' ').replace(/\s+(?:#\d+|\d{4,})$/,'') || 'Unknown';
function spending(transactions:Transaction[],month:string) {
  const seen=new Set<string>(); const categories=new Map<string,number>(); const merchants=new Map<string,{amount:number,count:number}>();
  let total=0; let count=0;
  for(const t of transactions) {
    if(monthOf(t.posted_at)!==month || t.currency!=='USD' || t.amount_cents>=0) continue;
    // Provider transaction IDs are stable within the selected connection. Never dedupe by amount/date.
    if(t.provider_external_id){if(seen.has(t.provider_external_id))continue;seen.add(t.provider_external_id);}
    const category=categoryOf(t.description);
    if(category==='Transfers'||category==='Refunds')continue;
    const amount=-t.amount_cents;total+=amount;count++;
    categories.set(category,(categories.get(category)||0)+amount);
    const merchant=merchantOf(t.description);const current=merchants.get(merchant)||{amount:0,count:0};
    merchants.set(merchant,{amount:current.amount+amount,count:current.count+1});
  }
  return {total,count,categories:[...categories].sort((a,b)=>b[1]-a[1]),merchants:[...merchants].sort((a,b)=>b[1].amount-a[1].amount)};
}
export function App(){
  const [provider,setProvider]=useState('northstar'),[connection,setConnection]=useState<Connection|null>(null);
  const [run,setRun]=useState<SyncRun|null>(null),[accounts,setAccounts]=useState<Account[]>([]),[txns,setTxns]=useState<Transaction[]>([]);
  const [busy,setBusy]=useState(false),[loading,setLoading]=useState(false),[error,setError]=useState('');
  const [month,setMonth]=useState('');
  // A response from an earlier connection must never overwrite the active institution.
  const activeConnectionId=useRef<string|null>(null);
  // Only the most recently requested connection may become active.
  const connectRequestId=useRef(0);
  // Later refreshes (for example, after a sync) must win over older requests for the same bank.
  const refreshRequestId=useRef(0);
  const months=useMemo(()=>[...new Set(txns.filter(t=>t.currency==='USD' && /^\d{4}-(0[1-9]|1[0-2])-\d{2}/.test(t.posted_at)).map(t=>monthOf(t.posted_at)))].sort().reverse(),[txns]);
  const selectedMonth=months.includes(month)?month:months[0]||'';
  const insights=useMemo(()=>spending(txns,selectedMonth),[txns,selectedMonth]);
  const previousMonth=selectedMonth?new Date(selectedMonth+'-01T00:00:00Z'):null;
  if(previousMonth)previousMonth.setUTCMonth(previousMonth.getUTCMonth()-1);
  const previousMonthKey=previousMonth?previousMonth.toISOString().slice(0,7):'';
  const hasPreviousHistory=months.includes(previousMonthKey);
  const previous=useMemo(()=>spending(txns,previousMonthKey),[txns,previousMonthKey]);
  const refresh=async(id:string)=>{const requestId=++refreshRequestId.current;setLoading(true);try{const [c,a,t]=await Promise.all([api.connection(id),api.accounts(id),api.transactions(id)]);if(activeConnectionId.current!==id || refreshRequestId.current!==requestId)return;setConnection(c);setAccounts(a.accounts||[]);setTxns(t.transactions||[])}finally{if(activeConnectionId.current===id && refreshRequestId.current===requestId)setLoading(false)}};
  useEffect(()=>{if(!run||!connection||!['requested','processing'].includes(run.status))return;
    let active=true;let polling=false;const timer=setInterval(async()=>{if(polling)return;polling=true;try{const next=await api.syncRun(run.id);if(!active || activeConnectionId.current!==connection.id)return;setRun(next);
      if(next.status==='succeeded'){clearInterval(timer);await refresh(connection.id)}
    }catch(e){if(active && activeConnectionId.current===connection.id)setError((e as Error).message)}finally{polling=false}},1000);
    return()=>{active=false;clearInterval(timer)};
  },[run,connection]);
  async function connect(e:FormEvent){e.preventDefault();const requestId=++connectRequestId.current;setBusy(true);setError('');try{
    const credentials:Record<string,string>=provider==='northstar'?{username:'demo',password:'demo'}:{token:'demo'};
    const c=await api.connect(provider,credentials);if(requestId!==connectRequestId.current)return;activeConnectionId.current=c.id;++refreshRequestId.current;setLoading(false);setConnection(c);setAccounts([]);setTxns([]);setRun(null);setMonth('');
    // Reconnecting to an existing mock institution should show persisted records immediately.
    await refresh(c.id);
  }catch(e){if(requestId===connectRequestId.current)setError((e as Error).message)}finally{if(requestId===connectRequestId.current)setBusy(false)}}
  async function sync(){if(!connection)return;const id=connection.id;setBusy(true);setError('');try{const next=await api.sync(id);if(activeConnectionId.current===id)setRun(next)}catch(e){if(activeConnectionId.current===id)setError((e as Error).message)}finally{setBusy(false)}}
  return <main><header><div><span className="eyebrow">FINANCIAL DATA CONNECTIVITY</span><h1>FinLink</h1><p>Connect a sandbox institution, synchronize transactions, and explore actual mock-bank records.</p></div><span className="badge">Sandbox · No live banks</span></header>
    <section className="panel connect"><div><h2>Institution connection</h2><p>Only synthetic demonstration credentials and data are used.</p></div><form onSubmit={connect}><label htmlFor="institution">Demo institution</label><select id="institution" value={provider} onChange={e=>setProvider(e.target.value)}><option value="northstar">Northstar Bank</option><option value="harbor">Harbor Credit Union</option></select><button type="submit" disabled={busy} aria-busy={busy}>{busy?'Connecting…':connection?'Connect another':'Connect institution'}</button></form></section>
    {error&&<div className="error" role="alert">{error}</div>}
    {connection&&<><section className="status"><div><span>Institution</span><strong>{connection.institution}</strong></div><div><span>Connection</span><strong>{connection.status}</strong></div><div><span>Last sync</span><strong>{connection.last_synced_at?new Date(connection.last_synced_at).toLocaleString():'Not synced'}</strong></div><button type="button" onClick={sync} disabled={busy||!!run&&['requested','processing'].includes(run.status)}>{busy?'Requesting…':'Sync now'}</button></section>
      {run&&<div className={`run ${run.status}`} role="status" aria-live="polite"><strong>Sync #{run.id}</strong><span>{run.status}</span>{run.status==='succeeded'&&<span>{run.account_count} accounts · {run.transaction_count} transactions</span>}{run.error_message&&<span>{run.error_message}</span>}</div>}
      {loading&&<p role="status">Loading synchronized records…</p>}
      <div className="grid" aria-busy={loading}><section className="panel"><h2>Accounts</h2>{accounts.length===0?<p className="empty">{loading?'Loading accounts…':'Sync to load account balances.'}</p>:accounts.map(a=><article className="account" key={a.provider_external_id}><div><strong>{a.name}</strong><span>{a.type}</span></div><b>{money(a.balance_cents,a.currency)}</b></article>)}</section>
      <section className="panel"><h2>Recent transactions</h2>{txns.length===0?<p className="empty">{loading?'Loading transactions…':'Sync to load transactions.'}</p>:<div className="transactions">{txns.map((t,i)=><article key={t.provider_external_id||i}><div><strong>{t.description}</strong><span>{new Date(t.posted_at).toLocaleDateString()}</span></div><b className={t.amount_cents<0?'debit':'credit'}>{money(t.amount_cents,t.currency)}</b></article>)}</div>}</section></div>
      <section className="panel" aria-labelledby="spending-heading"><h2 id="spending-heading">Spending insights</h2><p>Calculated from synchronized USD transactions. Transfers and labeled refunds are excluded; credits are not counted as spending.</p>{txns.length>=100&&<p className="error" role="status">Showing insights for the first 100 returned transactions only. Older transactions may be missing until pagination is supported; totals are not complete account history.</p>}
      {months.length===0?<p className="empty">No transaction history available yet. Run a sync to explore spending.</p>:<>
      <label htmlFor="spending-month">Month</label> <select id="spending-month" value={selectedMonth} onChange={e=>setMonth(e.target.value)}>{months.map(m=><option key={m} value={m}>{m}</option>)}</select>
      <div className="status"><div><span>Total spent</span><strong>{money(insights.total)}</strong></div><div><span>Purchases</span><strong>{insights.count}</strong></div><div><span>Previous month</span><strong>{hasPreviousHistory?money(previous.total):'No data'}</strong></div><div><span>Change</span><strong>{hasPreviousHistory&&previous.total?((insights.total-previous.total)/previous.total*100).toFixed(1)+'%':'Not enough history'}</strong></div></div>
      {insights.count===0?<p className="empty">No qualifying purchases in this month.</p>:<div className="grid"><div><h3>By category</h3>{insights.categories.map(([name,amount])=><div key={name} className="account"><span>{name}</span><strong>{money(amount)}</strong><progress aria-label={name+' share of spending'} value={amount} max={insights.total}/></div>)}</div><div><h3>Top merchants</h3>{insights.merchants.slice(0,5).map(([name,v])=><div className="account" key={name}><span>{name} · {v.count} payment{v.count===1?'':'s'}</span><strong>{money(v.amount)}</strong></div>)}</div></div>}
      </>}</section></>}
    <footer>Provider-independent data · Durable Kafka sync · Idempotent persistence · Mock institutions only</footer></main>;
}
