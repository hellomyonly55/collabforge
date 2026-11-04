<script setup lang="ts">
import {reactive,ref} from 'vue'
import {useRoute,useRouter} from 'vue-router'
import {useAuthStore} from '../../stores/auth'
import {errorMessage} from '../../api/client'
import {safeRedirect} from '../../utils/navigation'
import {passwordError} from '../../utils/validation'
import AuthLayout from '../../layouts/AuthLayout.vue'
const form=reactive({email:'',password:''}),busy=ref(false),error=ref('')
const auth=useAuthStore(),router=useRouter(),route=useRoute()
async function submit(){
 if(busy.value)return;error.value=passwordError(form.password);if(error.value)return
 busy.value=true
 try{await auth.login(form.email,form.password);form.password='';await router.replace(safeRedirect(route.query.redirect))}catch(e){error.value=errorMessage(e)}finally{busy.value=false}
}
</script>
<template><AuthLayout><span class="eyebrow">WELCOME BACK</span><h1>欢迎回来</h1><p class="auth-subtitle">登录 CollabForge，继续你的协作之旅。</p><el-alert v-if="route.query.registered && !error" title="注册成功，请登录" type="success" :closable="false" class="alert"/><el-alert v-if="route.query.reason==='password_updated' && !error" title="密码已更新，请重新登录" type="success" :closable="false" class="alert"/><el-alert v-if="error" :title="error" type="error" :closable="false" class="alert"/>
 <form class="auth-form" @submit.prevent="submit"><label for="login-email">邮箱</label><el-input id="login-email" v-model="form.email" type="email" autocomplete="username" placeholder="you@example.com" maxlength="254" required :disabled="busy"/><label for="login-password">密码</label><el-input id="login-password" v-model="form.password" type="password" show-password autocomplete="current-password" placeholder="输入你的密码" required :disabled="busy"/><el-button type="primary" native-type="submit" :loading="busy" class="submit-button">登录</el-button></form><p class="auth-switch">还没有账号？ <RouterLink to="/register">创建账号</RouterLink></p>
</AuthLayout></template>
