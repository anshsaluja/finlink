const BASE = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
export type Connection={id:string;provider:string;institution:string;status:string;created_at:string;last_synced_at?:string;cursor?:string};
export type Account={provider_external_id:string;name:string;type:string;currency:string;balance_cents:number};
export type Transaction={provider_external_id:string;description:string;amount_cents:number;currency:string;posted_at:string};
export type SyncRun={id:number;status:string;account_count:number;transaction_count:number;error_message?:string};
async function request<T>(path:string,init?:RequestInit):Promise<T>{const r=await fetch(`${BASE}${path}`,{...init,headers:{'Content-Type':'application/json',...(init?.headers||{})}});if(!r.ok){const e=await r.json().catch(()=>({message:r.statusText}));throw new Error(e.message||`Request failed (${r.status})`)}return r.json()}
export const api={
 connect:(provider:string,credentials:Record<string,string>)=>request<Connection>('/v1/connections',{method:'POST',body:JSON.stringify({provider,credentials})}),
 connection:(id:string)=>request<Connection>(`/v1/connections/${id}`),
 sync:(id:string)=>request<SyncRun>(`/v1/connections/${id}/sync`,{method:'POST',headers:{'Idempotency-Key':crypto.randomUUID()}}),
 syncRun:(id:number)=>request<SyncRun>(`/v1/sync-runs/${id}`),
 accounts:(id:string)=>request<{accounts:Account[]}>(`/v1/connections/${id}/accounts`),
 transactions:(id:string,offset=0,limit=200)=>request<{transactions:Transaction[];limit:number;offset:number;has_more?:boolean}>(`/v1/connections/${id}/transactions?limit=${limit}&offset=${offset}`),
 // A full page is only a continuation hint; load every page before calculating totals.
 async allTransactions(id:string):Promise<Transaction[]>{
   const result:Transaction[]=[];const seen=new Set<string>();const pageSize=200;
   for(let offset=0;offset<=1_000_000;offset+=pageSize){
     const page=await request<{transactions:Transaction[];limit:number;offset:number}>(`/v1/connections/${id}/transactions?limit=${pageSize}&offset=${offset}`);
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
     for(const record of records){
       const key=record.provider_external_id;
       if(key){if(seen.has(key))continue;seen.add(key);}
       result.push(record);
     }
     if(records.length<pageSize){
       // Offset pages can shift if a sync inserts new rows during retrieval.
       // Recheck the newest page before trusting the resulting analytics.
       if(offset>0){
         const head=await request<{transactions:Transaction[];limit:number;offset:number}>(`/v1/connections/${id}/transactions?limit=${pageSize}&offset=0`);
         if(head.limit!==pageSize || head.offset!==0 || !Array.isArray(head.transactions) || head.transactions.length!==pageSize){
           throw new Error('Transaction history changed during retrieval; retry to verify spending totals.');
         }
         const firstIds=result.slice(0,pageSize).map(t=>t.provider_external_id);
         if(head.transactions.some((t,i)=>!t || t.provider_external_id!==firstIds[i])){
           throw new Error('Transaction history changed during retrieval; retry to verify spending totals.');
         }
       }
       return result;
     }
   }
   throw new Error('Transaction history exceeds the supported pagination window; spending totals cannot be verified.');
 }
};
