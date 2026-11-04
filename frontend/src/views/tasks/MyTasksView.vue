<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import PaginationBar from "../../components/PaginationBar.vue";
import TaskFiltersPanel from "../../components/TaskFilters.vue";
import { taskAPI } from "../../api/tasks";
import { errorMessage } from "../../api/client";
import type { TaskPage } from "../../types/task";
import {
  emptyTaskFilters,
  statusLabels,
  priorityLabels,
} from "../../utils/task";
const filters = ref(emptyTaskFilters()),
  applied = ref(emptyTaskFilters()),
  data = ref<TaskPage | null>(null),
  page = ref(1),
  loading = ref(false),
  error = ref("");
let generation = 0;
onBeforeUnmount(() => generation++);
async function load(next = page.value) {
  const current = ++generation;
  page.value = next;
  loading.value = true;
  error.value = "";
  data.value = null;
  try {
    const result = (
      await taskAPI.mine(
        { ...applied.value, assignee_id: undefined },
        page.value,
      )
    ).data;
    if (current === generation) data.value = result;
  } catch (e) {
    if (current === generation) error.value = errorMessage(e);
  } finally {
    if (current === generation) loading.value = false;
  }
}
function apply() {
  applied.value = { ...filters.value };
  void load(1);
}
function reset() {
  filters.value = emptyTaskFilters();
  apply();
}
onMounted(() => load());
</script>
<template>
  <WorkspaceLayout
    ><div class="eyebrow">MY TASKS</div>
    <div class="team-page-heading">
      <div>
        <h1 class="workspace-title">我的任务</h1>
        <p>查看分配给你的真实任务，按截止日期排列。</p>
      </div>
      <el-button :loading="loading" @click="load()">刷新任务</el-button>
    </div>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      class="alert" /><TaskFiltersPanel
      v-model="filters"
      mine
      :busy="loading"
      @apply="apply"
      @reset="reset" />
    <div v-if="loading" class="team-empty" role="status">正在加载任务…</div>
    <section v-else-if="data" class="team-panel">
      <div class="table-scroll">
        <table class="team-table task-table">
          <thead>
            <tr>
              <th>任务</th>
              <th>项目 / 团队</th>
              <th>状态</th>
              <th>优先级</th>
              <th>截止日期</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="task in data.items" :key="task.id">
              <td>
                <RouterLink :to="`/tasks/${task.id}`" class="task-title-link">{{
                  task.title
                }}</RouterLink>
                <p v-if="task.project_archived_at || task.team_archived_at">
                  已归档 · 仅可查看
                </p>
              </td>
              <td>
                <RouterLink :to="`/projects/${task.project_id}/board`">{{
                  task.project_name
                }}</RouterLink>
                <p>{{ task.team_name }}</p>
              </td>
              <td>{{ statusLabels[task.status] }}</td>
              <td>{{ priorityLabels[task.priority] }}</td>
              <td :class="{ 'task-overdue': task.is_overdue }">
                {{ task.due_at || "未设置" }}
              </td>
            </tr>
            <tr v-if="!data.items.length">
              <td colspan="5" class="table-empty">
                暂无分配给你的符合条件的任务
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <PaginationBar
        :page="page"
        :total="data.total"
        :busy="loading"
        @change="load"
      /></section
  ></WorkspaceLayout>
</template>
