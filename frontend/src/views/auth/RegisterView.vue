<script setup lang="ts">
import {reactive,ref} from 'vue'
import {useRouter} from 'vue-router'
import {authAPI} from '../../api/auth'
import {errorMessage} from '../../api/client'
import {nicknameError,passwordError} from '../../utils/validation'
import AuthLayout from '../../layouts/AuthLayout.vue'
const form=reactive({email:'',nickname:'',password:'',confirm:''}),busy=ref(false),error=ref(''),router=useRouter()
async function submit(){
 if(busy.value)return
 error.value=nicknameError(form.nickname)||passwordError(form.password)||(form.password!==form.confirm?'两次输入的密码不一致':'');if(error.value)return
 busy.value=true
 try{await authAPI.register({email:form.email,nickname:form.nickname,password:form.password});form.password='';form.confirm='';await router.replace({path:'/login',query:{registered:'1'}})}catch(e){error.value=errorMessage(e)}finally{busy.value=false}
}
</script>
<template><AuthLayout><span class="eyebrow">START YOUR FLOW</span><h1>创建你的账号</h1><p class="auth-subtitle">从一个账号开始，连接团队与想法。</p><el-alert v-if="error" :title="error" type="error" :closable="false" class="alert"/><form class="auth-form" @submit.prevent="submit"><label for="register-nickname">昵称</label><el-input id="register-nickname" v-model="form.nickname" autocomplete="nickname" placeholder="团队伙伴如何称呼你" required :disabled="busy"/><label for="register-email">邮箱</label><el-input id="register-email" v-model="form.email" type="email" autocomplete="username" placeholder="you@example.com" maxlength="254" required :disabled="busy"/><label for="register-password">密码</label><el-input id="register-password" v-model="form.password" type="password" show-password autocomplete="new-password" placeholder="至少 8 个字符，最多 72 字节" required :disabled="busy"/><label for="register-confirm">确认密码</label><el-input id="register-confirm" v-model="form.confirm" type="password" show-password autocomplete="new-password" placeholder="再次输入密码" required :disabled="busy"/><el-button type="primary" native-type="submit" :loading="busy" class="submit-button">创建账号</el-button></form><p class="auth-switch">已有账号？ <RouterLink to="/login">前往登录</RouterLink></p></AuthLayout></template>
