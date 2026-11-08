const BASE = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
export type Connection={id:string;provider:string;institution:string;status:string;created_at:string;last_synced_at?:string;cursor?:string};
export type Account={provider_external_id:string;name:string;type:string;currency:string;balance_cents:number};
export type Transaction={provider_external_id:string;description:string;amount_cents:number;currency:string;posted_at:string};
export type SyncRun={id:number;status:string;account_count:number;transaction_count:number;error_message?:string};
async function request<T>(path:string,init?:RequestInit):Promise<T>{const r=await fetch(`${BASE}${path}`,{...init,headers:{'Content-Type':'application/json',...(init?.headers||{})}});if(!r.ok){const e=await r.json().catch(()=>({message:r.statusText}));throw new Error(e.message||`Request failed (${r.status})`)}return r.json()}
// Require an unambiguous ISO timestamp with an explicit timezone. Date.parse alone
// silently normalizes invalid calendar dates (for example, February 30).
export function validPostedAt(value:string):boolean {
  const match=/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?(Z|[+-]\d{2}:\d{2})$/.exec(value);
  if(!match)return false;
  const [,year,month,day,hour,minute,second,zone]=match;
  const y=Number(year),m=Number(month),d=Number(day),h=Number(hour),min=Number(minute),sec=Number(second);
  if(m<1||m>12||d<1||h>23||min>59||sec>59)return false;
  const daysInMonth=new Date(Date.UTC(y,m,0)).getUTCDate();
  if(d>daysInMonth)return false;
  if(zone!=='Z'){
    const zh=Number(zone.slice(1,3)),zm=Number(zone.slice(4,6));
    if(zh>23||zm>59)return false;
  }
  return Number.isFinite(Date.parse(value));
}
const connectionPath=(id:string)=>`/v1/connections/${encodeURIComponent(id)}`;
export const api={
 connect:(provider:string,credentials:Record<string,string>)=>request<Connection>('/v1/connections',{method:'POST',body:JSON.stringify({provider,credentials})}),
 connection:(id:string)=>request<Connection>(`${connectionPath(id)}`),
 sync:(id:string)=>request<SyncRun>(`${connectionPath(id)}/sync`,{method:'POST',headers:{'Idempotency-Key':crypto.randomUUID()}}),
 syncRun:(id:number)=>request<SyncRun>(`/v1/sync-runs/${id}`),
 accounts:(id:string)=>request<{accounts:Account[]}>(`${connectionPath(id)}/accounts`),
 transactions:(id:string,offset=0,limit=200)=>request<{transactions:Transaction[];limit:number;offset:number;has_more?:boolean}>(`${connectionPath(id)}/transactions?limit=${limit}&offset=${offset}`),
 // A full page is only a continuation hint; load every page before calculating totals.
 async allTransactions(id:string):Promise<Transaction[]>{
   const result:Transaction[]=[];const seen=new Set<string>();const pageSize=200;
   for(let offset=0;offset<=1_000_000;offset+=pageSize){
     const page=await request<{transactions:Transaction[];limit:number;offset:number;has_more?:boolean}>(`${connectionPath(id)}/transactions?limit=${pageSize}&offset=${offset}`);
     if(page.limit!==pageSize || page.offset!==offset){
       throw new Error('Transaction page metadata does not match the requested window; spending totals cannot be verified.');
     }
     const records=page.transactions;
     if(!Array.isArray(records) || records.length>pageSize){
       throw new Error('Invalid transaction page returned by the API; spending totals cannot be verified.');
     }
     if(records.some(record=>!record || typeof record.provider_external_id!=='string' || !record.provider_external_id || typeof record.amount_cents!=='number' || !Number.isSafeInteger(record.amount_cents) || typeof record.currency!=='string' || typeof record.description!=='string' || typeof record.posted_at!=='string')){
       throw new Error('Malformed transaction record returned by the API; spending totals cannot be verified.');
     }
     if(records.some(record=>!validPostedAt(record.posted_at))){
       throw new Error('Invalid transaction timestamp; spending totals cannot be verified.');
     }
     if(typeof page.has_more!=='undefined' && typeof page.has_more!=='boolean'){
       throw new Error('Invalid transaction continuation metadata; spending totals cannot be verified.');
     }
     if(records.length<pageSize && page.has_more===true){
       throw new Error('Transaction page is incomplete despite a continuation hint; spending totals cannot be verified.');
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
