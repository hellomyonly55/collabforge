export class APIError extends Error {
  constructor(message:string, public status:number, public requestId?:string, public code?:string){super(message)}
}
interface Options {method?:'GET'|'POST'|'PATCH'|'PUT'|'DELETE';body?:unknown;acceptedStatuses?:number[]}
export async function request<T>(path:string, options:Options={}):Promise<T>{
  let response:Response
  try {
    response=await fetch(`/api/v1${path}`,{method:options.method??'GET',credentials:'same-origin',signal:AbortSignal.timeout(10000),headers:{Accept:'application/json',...(options.body!==undefined?{'Content-Type':'application/json'}:{})},body:options.body===undefined?undefined:JSON.stringify(options.body)})
  }catch{throw new APIError('无法连接后端，请检查服务是否启动',0)}
  if(response.status===204)return undefined as T
  let body:unknown
  try{body=await response.json()}catch{throw new APIError('后端未返回有效 JSON，请检查代理配置',response.status)}
  if(!response.ok&&!options.acceptedStatuses?.includes(response.status)){
    const b=body as {error?:{message?:string;code?:string};request_id?:string}
    if(response.status===401&&!path.startsWith('/auth/'))window.dispatchEvent(new Event('taskflow:unauthorized'))
    throw new APIError(b.error?.message??'请求失败',response.status,b.request_id,b.error?.code)
  }
  return body as T
}
export function errorMessage(error:unknown):string{
  if(error instanceof APIError)return error.message+(error.requestId?`（请求编号：${error.requestId}）`:'')
  return '操作失败，请稍后重试'
}
