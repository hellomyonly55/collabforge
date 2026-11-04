<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "../stores/auth";
import { errorMessage } from "../api/client";
const auth = useAuthStore(),
  router = useRouter(),
  busy = ref(false),
  error = ref("");
async function logout() {
  busy.value = true;
  error.value = "";
  try {
    await auth.logout();
    await router.replace("/login");
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <div class="workspace-shell">
    <aside class="workspace-aside">
      <RouterLink to="/teams" class="brand"
        ><span class="logo">C</span>CollabForge</RouterLink
      >
      <div class="workspace">个人工作区</div>
      <nav>
        <RouterLink
          to="/teams"
          :class="{ 'router-link-active': $route.meta.section === 'projects' }"
          >◈ <span>团队空间</span></RouterLink
        ><RouterLink
          to="/my-tasks"
          aria-label="我的任务"
          :class="{ 'router-link-active': $route.meta.section === 'tasks' }"
          >▣ <span>我的任务</span></RouterLink
        ><RouterLink to="/settings">◎ <span>个人设置</span></RouterLink
        ><RouterLink
          v-if="auth.user?.platform_role === 'admin'"
          to="/admin"
          aria-label="管理员后台"
          >▤ <span>管理员后台</span></RouterLink
        ><RouterLink to="/health">◉ <span>系统状态</span></RouterLink>
      </nav>
      <p class="aside-note">一步一步，把协作变简单。</p>
    </aside>
    <main class="workspace-main">
      <header>
        <span>{{
          $route.meta.section === "admin"
            ? "平台管理"
            : $route.meta.section === "projects"
              ? "项目管理"
              : $route.meta.section === "tasks"
                ? "我的任务"
                : $route.path === "/settings"
                  ? "个人设置"
                  : "团队空间"
        }}</span>
        <div class="user-tools">
          <span class="user-avatar">{{ auth.user?.nickname.slice(0, 1) }}</span
          ><span>{{ auth.user?.nickname }}</span
          ><el-button text :loading="busy" @click="logout">退出登录</el-button>
        </div>
      </header>
      <div class="workspace-content">
        <el-alert
          v-if="error"
          :title="error"
          type="error"
          :closable="false"
          class="alert"
        /><slot />
      </div>
    </main>
  </div>
</template>
