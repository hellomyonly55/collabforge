<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import TaskPanel from "../../components/TaskPanel.vue";
const route = useRoute(),
  router = useRouter(),
  id = computed(() => String(route.params.taskId));
</script>
<template>
  <WorkspaceLayout
    ><RouterLink to="/my-tasks" class="back-link">← 返回我的任务</RouterLink>
    <div class="eyebrow">TASK DETAIL</div>
    <h1 class="workspace-title">任务详情</h1>
    <section class="team-panel task-detail-page">
      <TaskPanel
        :task-i-d="id"
        @deleted="(projectID) => router.replace(`/projects/${projectID}/board`)"
      /></section
  ></WorkspaceLayout>
</template>
