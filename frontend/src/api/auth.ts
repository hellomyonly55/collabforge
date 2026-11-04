import {request} from './client'
import type {Envelope,User} from '../types/api'
export const authAPI={
 me:()=>request<Envelope<User>>('/me'),
 register:(data:{email:string;nickname:string;password:string})=>request<Envelope<User>>('/auth/register',{method:'POST',body:data}),
 login:(data:{email:string;password:string})=>request<Envelope<User>>('/auth/login',{method:'POST',body:data}),
 logout:()=>request<void>('/auth/logout',{method:'POST'}),
 profile:(nickname:string)=>request<Envelope<User>>('/me',{method:'PATCH',body:{nickname}}),
 password:(old_password:string,new_password:string)=>request<void>('/me/password',{method:'PUT',body:{old_password,new_password}}),
}
