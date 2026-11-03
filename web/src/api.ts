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
 transactions:(id:string)=>request<{transactions:Transaction[]}>(`/v1/connections/${id}/transactions?limit=100`)
};
