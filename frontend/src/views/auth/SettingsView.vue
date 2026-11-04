<script setup lang="ts">
import {reactive,ref,watch} from 'vue'
import {useRouter} from 'vue-router'
import WorkspaceLayout from '../../layouts/WorkspaceLayout.vue'
import {useAuthStore} from '../../stores/auth'
import {errorMessage} from '../../api/client'
import {nicknameError,passwordError} from '../../utils/validation'
const auth=useAuthStore(),router=useRouter(),name=ref(auth.user?.nickname??''),profileBusy=ref(false),passwordBusy=ref(false),profileError=ref(''),passwordMessage=ref(''),saved=ref(false)
const form=reactive({oldPassword:'',newPassword:'',confirm:''})
watch(()=>auth.user?.nickname,value=>{if(value!==undefined)name.value=value})
async function saveProfile(){
 if(profileBusy.value)return;saved.value=false;profileError.value=nicknameError(name.value);if(profileError.value)return
 profileBusy.value=true;try{await auth.updateProfile(name.value);saved.value=true}catch(e){profileError.value=errorMessage(e)}finally{profileBusy.value=false}
}
async function changePassword(){
 if(passwordBusy.value)return
 passwordMessage.value=passwordError(form.newPassword)||(form.newPassword!==form.confirm?'两次输入的新密码不一致':'');if(passwordMessage.value)return
 passwordBusy.value=true
 try{await auth.changePassword(form.oldPassword,form.newPassword);form.oldPassword='';form.newPassword='';form.confirm='';await router.replace({path:'/login',query:{reason:'password_updated'}})}catch(e){passwordMessage.value=errorMessage(e)}finally{passwordBusy.value=false}
}
</script>
<template><WorkspaceLayout><div class="eyebrow">ACCOUNT SETTINGS</div><h1 class="workspace-title">个人设置</h1><p>维护个人资料，让你的团队更了解你。</p><div class="settings-grid"><section class="settings-card"><h2>个人资料</h2><p>邮箱作为登录账号，当前版本暂不支持更改。</p><el-alert v-if="profileError" :title="profileError" type="error" :closable="false" class="alert"/><el-alert v-if="saved" title="个人资料已保存" type="success" :closable="false" class="alert"/><form class="auth-form" @submit.prevent="saveProfile"><label for="profile-email">邮箱</label><el-input id="profile-email" :model-value="auth.user?.email" disabled/><label for="profile-nickname">昵称</label><el-input id="profile-nickname" v-model="name" autocomplete="nickname" required :disabled="profileBusy"/><p class="field-hint">1 至 40 个字符</p><el-button type="primary" native-type="submit" :loading="profileBusy">保存资料</el-button></form></section>
 <section class="settings-card"><h2>修改密码</h2><p>修改成功后，所有已登录设备都需要重新登录。</p><el-alert v-if="passwordMessage" :title="passwordMessage" type="error" :closable="false" class="alert"/><form class="auth-form" @submit.prevent="changePassword"><label for="old-password">原密码</label><el-input id="old-password" v-model="form.oldPassword" type="password" show-password autocomplete="current-password" required :disabled="passwordBusy"/><label for="new-password">新密码</label><el-input id="new-password" v-model="form.newPassword" type="password" show-password autocomplete="new-password" placeholder="至少 8 个字符，最多 72 字节" required :disabled="passwordBusy"/><label for="confirm-password">确认新密码</label><el-input id="confirm-password" v-model="form.confirm" type="password" show-password autocomplete="new-password" required :disabled="passwordBusy"/><el-button type="primary" native-type="submit" :loading="passwordBusy">更新密码</el-button></form></section></div></WorkspaceLayout></template>
