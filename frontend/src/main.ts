import {createApp} from 'vue'
import {createPinia} from 'pinia'
import {ElAlert, ElButton, ElTag, ElInput, ElDrawer} from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import {router} from './router'
import './style.css'
import {useAuthStore} from './stores/auth'
const app=createApp(App)
app.use(createPinia())
window.addEventListener('taskflow:unauthorized',()=>{
 useAuthStore().clear()
 if(router.currentRoute.value.meta.requiresAuth)void router.replace({path:'/login',query:{redirect:router.currentRoute.value.path}})
})
app.use(router).use(ElAlert).use(ElButton).use(ElTag).use(ElInput).use(ElDrawer).mount('#app')
