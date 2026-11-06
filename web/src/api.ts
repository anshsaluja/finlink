const BASE = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
export type Connection={id:string;provider:string;institution:string;status:string;created_at:string;last_synced_at?:string;cursor?:string};
export type Account={provider_external_id:string;name:string;type:string;currency:string;balance_cents:number};
export type Transaction={provider_external_id:string;description:string;amount_cents:number;currency:string;posted_at:string};
// Reject calendar rollovers and invalid timezone offsets that Date.parse may normalize.
export function isValidTransactionTimestamp(value:string):boolean {
 const match=/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d+)?(Z|([+-])(\d{2}):(\d{2}))$/.exec(value);
 if(!match) return false;
 const [,year,month,day,hour,minute,second,_zone,_sign,offsetHour,offsetMinute]=match;
 const y=Number(year),m=Number(month),d=Number(day),h=Number(hour),min=Number(minute),sec=Number(second);
 if(y<1 || m<1 || m>12 || d<1 || h>23 || min>59 || sec>59) return false;
 // Date.UTC treats years 0-99 as 1900-1999, so calculate leap years explicitly.
 const leap=y%4===0 && (y%100!==0 || y%400===0);
 const maxDay=m===2?(leap?29:28):([4,6,9,11].includes(m)?30:31);
 if(d>maxDay || (offsetHour!==undefined && (Number(offsetHour)>23 || Number(offsetMinute)>59))) return false;
 return Number.isFinite(Date.parse(value));
}
export type SyncRun={id:number;status:string;account_count:number;transaction_count:number;error_message?:string};
async function request<T>(path:string,init?:RequestInit):Promise<T>{const r=await fetch(`${BASE}${path}`,{...init,headers:{'Content-Type':'application/json',...(init?.headers||{})}});if(!r.ok){const e=await r.json().catch(()=>({message:r.statusText}));throw new Error(e.message||`Request failed (${r.status})`)}return r.json()}
// Connection identifiers are untrusted URL path segments, never route fragments.
const connectionPath=(id:string)=>`/v1/connections/${encodeURIComponent(id)}`;
export const api={
 connect:(provider:string,credentials:Record<string,string>)=>request<Connection>('/v1/connections',{method:'POST',body:JSON.stringify({provider,credentials})}),
 connection:(id:string)=>request<Connection>(connectionPath(id)),
 sync:(id:string)=>request<SyncRun>(`${connectionPath(id)}/sync`,{method:'POST',headers:{'Idempotency-Key':crypto.randomUUID()}}),
 syncRun:(id:number)=>request<SyncRun>(`/v1/sync-runs/${id}`),
 accounts:(id:string)=>request<{accounts:Account[]}>(`${connectionPath(id)}/accounts`),
 transactions:(id:string,offset=0,limit=200)=>request<{transactions:Transaction[];limit:number;offset:number;has_more?:boolean}>(`${connectionPath(id)}/transactions?limit=${limit}&offset=${offset}`),
 // A full page is only a continuation hint; load every page before calculating totals.
 async allTransactions(id:string):Promise<Transaction[]>{
   const result:Transaction[]=[];const seen=new Set<string>();const pageSize=200;
   for(let offset=0;offset<=1_000_000;offset+=pageSize){
     const page=await request<{transactions:Transaction[];limit:number;offset:number}>(`${connectionPath(id)}/transactions?limit=${pageSize}&offset=${offset}`);
     if(page.limit!==pageSize || page.offset!==offset){
       throw new Error('Transaction page metadata does not match the requested window; spending totals cannot be verified.');
     }
     const records=page.transactions;
     if(!Array.isArray(records) || records.length>pageSize){
       throw new Error('Invalid transaction page returned by the API; spending totals cannot be verified.');
     }
     if(records.some(record=>!record || typeof record.provider_external_id!=='string' || !record.provider_external_id || typeof record.amount_cents!=='number' || !Number.isSafeInteger(record.amount_cents) || typeof record.currency!=='string' || typeof record.description!=='string' || typeof record.posted_at!=='string' || !isValidTransactionTimestamp(record.posted_at))){
       throw new Error('Malformed transaction record returned by the API; spending totals cannot be verified.');
     }
     for(const record of records){
       const key=record.provider_external_id;
       if(seen.has(key)){
         throw new Error('Duplicate transaction ID across pages; spending totals cannot be verified.');
       }
       seen.add(key);
       result.push(record);
     }
     if(records.length<pageSize){
       // Offset pages can shift if a sync inserts new rows during retrieval.
       // Recheck the newest page before trusting the resulting analytics.
       if(offset>0){
         const head=await request<{transactions:Transaction[];limit:number;offset:number}>(`${connectionPath(id)}/transactions?limit=${pageSize}&offset=0`);
         if(head.limit!==pageSize || head.offset!==0 || !Array.isArray(head.transactions) || head.transactions.length!==pageSize){
           throw new Error('Transaction history changed during retrieval; retry to verify spending totals.');
         }
         const firstPage=result.slice(0,pageSize);
         if(head.transactions.some((t,i)=>!t || t.provider_external_id!==firstPage[i].provider_external_id || t.amount_cents!==firstPage[i].amount_cents || t.currency!==firstPage[i].currency || t.description!==firstPage[i].description || t.posted_at!==firstPage[i].posted_at)){
           throw new Error('Transaction history changed during retrieval; retry to verify spending totals.');
         }
       }
       return result;
     }
   }
   throw new Error('Transaction history exceeds the supported pagination window; spending totals cannot be verified.');
 }
};
