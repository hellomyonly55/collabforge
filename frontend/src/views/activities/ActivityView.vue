<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useRoute } from "vue-router";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import ActivityFeed from "../../components/ActivityFeed.vue";
import { teamAPI } from "../../api/teams";
import { projectAPI } from "../../api/projects";
import { errorMessage } from "../../api/client";
const route = useRoute(),
  scope = computed((): "teams" | "projects" =>
    route.params.teamId ? "teams" : "projects",
  ),
  id = computed(() => String(route.params.teamId ?? route.params.projectId)),
  name = ref(""),
  error = ref(""),
  loading = ref(false);
let generation = 0;
onBeforeUnmount(() => generation++);
async function load() {
  const current = ++generation;
  loading.value = true;
  name.value = "";
  error.value = "";
  try {
    const data =
      scope.value === "teams"
        ? (await teamAPI.get(id.value)).data
        : (await projectAPI.get(id.value)).data;
    if (current === generation) name.value = data.name;
  } catch (e) {
    if (current === generation) error.value = errorMessage(e);
  } finally {
    if (current === generation) loading.value = false;
  }
}
watch([scope, id], load, { immediate: true });
</script>
<template>
  <WorkspaceLayout
    ><RouterLink :to="`/${scope}/${id}`" class="back-link"
      >← 返回{{ scope === "teams" ? "团队" : "项目" }}</RouterLink
    >
    <div class="eyebrow">ACTIVITY LOG</div>
    <h1 class="workspace-title">操作日志</h1>
    <p class="workspace-subtitle">{{ name }}</p>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      class="alert" />
    <p v-if="loading" role="status">正在加载…</p>
    <el-button v-if="error" @click="load">重试</el-button>
    <section v-if="name && !loading" class="team-panel">
      <ActivityFeed
        :scope="scope"
        :resource-i-d="id"
        @unavailable="load"
      /></section
  ></WorkspaceLayout>
</template>
