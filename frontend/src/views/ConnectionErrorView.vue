<script setup lang="ts">
import {ref} from 'vue'
import {useRoute,useRouter} from 'vue-router'
import {useAuthStore} from '../stores/auth'
import {safeRedirect} from '../utils/navigation'
import {errorMessage} from '../api/client'
const route=useRoute(),router=useRouter(),auth=useAuthStore(),error=ref(''),busy=ref(false)
async function retry(){busy.value=true;error.value='';try{await auth.refresh();await router.replace(auth.user?safeRedirect(route.query.redirect):'/login')}catch(e){error.value=errorMessage(e)}finally{busy.value=false}}
</script>
<template><div class="connection-page"><span class="logo">T</span><h1>暂时无法连接工作区</h1><p>账号验证服务未就绪，请检查后端、MySQL 和 Redis 状态后重试。</p><el-alert v-if="error" :title="error" type="error" :closable="false"/><el-button type="primary" :loading="busy" @click="retry">重新连接</el-button><RouterLink to="/health">查看系统状态</RouterLink></div></template>
