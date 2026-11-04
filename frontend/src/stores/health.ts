import {defineStore} from 'pinia'
import {ref} from 'vue'
import {APIError,request} from '../api/client'
import type {Envelope,Health} from '../types/api'
export const useHealthStore=defineStore('health',()=>{
 const loading=ref(false),health=ref<Health|null>(null),error=ref(''),requestId=ref(''),checkedAt=ref('')
 async function refresh(){
  if(loading.value)return
  loading.value=true;error.value='';health.value=null;requestId.value='';checkedAt.value=''
  try{
   // Readiness returns 503 with dependency details, which are real diagnostics rather than fake success data.
   const body=await request<Envelope<Health>>('/health/ready',{acceptedStatuses:[503]})
   if(!body.data?.checks){throw new APIError('健康检查返回异常响应',502,body.request_id)}
   health.value=body.data;requestId.value=body.request_id
   checkedAt.value=new Date().toLocaleString('zh-CN',{hour12:false})
  }catch(e){error.value=e instanceof APIError?e.message:'无法获取健康状态，请检查后端和代理是否启动'}finally{loading.value=false}
 }
 return {loading,health,error,requestId,checkedAt,refresh}
})
