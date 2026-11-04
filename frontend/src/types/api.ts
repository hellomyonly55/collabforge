export interface Envelope<T>{data:T;request_id:string}
export interface Health {status:'alive'|'ready'|'not_ready';checks?:Record<string,'ok'|'unavailable'>}
export interface User {id:string;email:string;nickname:string;platform_role:'user'|'admin';created_at:string}
