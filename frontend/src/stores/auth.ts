import {defineStore} from 'pinia'
import {ref} from 'vue'
import {authAPI} from '../api/auth'
import {APIError} from '../api/client'
import type {User} from '../types/api'
export const useAuthStore=defineStore('auth',()=>{
  const user=ref<User|null>(null),checked=ref(false)
  function clear(){user.value=null;checked.value=true}
  async function refresh(){
    try{user.value=(await authAPI.me()).data;checked.value=true}
    catch(e){if(e instanceof APIError&&e.status===401){clear();return}throw e}
  }
  async function login(email:string,password:string){user.value=(await authAPI.login({email,password})).data;checked.value=true}
  async function logout(){await authAPI.logout();clear()}
  async function updateProfile(nickname:string){user.value=(await authAPI.profile(nickname)).data}
  async function changePassword(oldPassword:string,newPassword:string){await authAPI.password(oldPassword,newPassword);clear()}
  return {user,checked,clear,refresh,login,logout,updateProfile,changePassword}
})
