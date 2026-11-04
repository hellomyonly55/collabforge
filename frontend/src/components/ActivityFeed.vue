<script setup lang="ts">
import { onBeforeUnmount, reactive, ref, watch } from "vue";
import { listActivities, emptyActivityFilters } from "../api/activities";
import { APIError, errorMessage } from "../api/client";
import type { Page, Activity } from "../types/team";
import { activityLabels, dateTime, roleLabels } from "../utils/team";
import { statusLabels } from "../utils/task";
import PaginationBar from "./PaginationBar.vue";
const props = defineProps<{
  scope: "teams" | "projects";
  resourceID: string;
}>();
const emit = defineEmits<{ unavailable: [] }>();
const filters = reactive(emptyActivityFilters()),
  applied = ref(emptyActivityFilters()),
  page = ref(1),
  result = ref<Page<Activity> | null>(null),
  loading = ref(false),
  error = ref("");
const targetLabels: Record<string, string> = {
  team: "团队",
  user: "成员",
  invite: "邀请",
  project: "项目",
  task: "任务",
  comment: "评论",
};
let generation = 0;
onBeforeUnmount(() => generation++);
async function load() {
  const current = ++generation;
  loading.value = true;
  error.value = "";
  result.value = null;
  try {
    const data = (
      await listActivities(
        props.scope,
        props.resourceID,
        applied.value,
        page.value,
      )
    ).data;
    if (current === generation) result.value = data;
  } catch (e) {
    if (current === generation) {
      error.value = errorMessage(e);
      if (e instanceof APIError && [403, 404].includes(e.status))
        emit("unavailable");
    }
  } finally {
    if (current === generation) loading.value = false;
  }
}
watch(
  [() => props.scope, () => props.resourceID],
  () => {
    Object.assign(filters, emptyActivityFilters());
    applied.value = emptyActivityFilters();
    page.value = 1;
    void load();
  },
  { immediate: true },
);
function search() {
  if (filters.from && filters.to && filters.from > filters.to) {
    error.value = "开始日期不能晚于结束日期";
    return;
  }
  applied.value = { ...filters };
  page.value = 1;
  void load();
}
function reset() {
  Object.assign(filters, emptyActivityFilters());
  search();
}
function changePage(value: number) {
  page.value = value;
  void load();
}
function taskID(a: Activity): string | null {
  const id =
    a.target_type === "task"
      ? a.target_id
      : a.target_type === "comment"
        ? a.metadata.task_id
        : null;
  return typeof id === "string" && /^[1-9][0-9]{0,19}$/.test(id) ? id : null;
}
function text(a: Activity): string {
  const v = a.metadata.title ?? a.metadata.task_title ?? a.metadata.name;
  return typeof v === "string" ? v : "";
}
function transition(a: Activity): string {
  const from = a.metadata.from,
    to = a.metadata.to;
  if (typeof from !== "string" || typeof to !== "string") return "";
  const names: Record<string, string> =
    a.target_type === "task" ? statusLabels : roleLabels;
  return `${names[from] ?? from} → ${names[to] ?? to}`;
}
</script>
<template>
  <div class="activity-feed">
    <form
      class="activity-filters"
      aria-label="日志筛选"
      @submit.prevent="search"
    >
      <div>
        <label for="log-action">操作类型</label
        ><select id="log-action" v-model="filters.action" :disabled="loading">
          <option value="">全部操作</option>
          <option
            v-for="(label, key) in activityLabels"
            :key="key"
            :value="key"
          >
            {{ label }}
          </option>
        </select>
      </div>
      <div>
        <label for="log-target">目标类型</label
        ><select
          id="log-target"
          v-model="filters.target_type"
          :disabled="loading"
        >
          <option value="">全部目标</option>
          <option v-for="(label, key) in targetLabels" :key="key" :value="key">
            {{ label }}
          </option>
        </select>
      </div>
      <div>
        <label for="log-actor">操作者 ID</label
        ><input
          id="log-actor"
          v-model="filters.actor_id"
          inputmode="numeric"
          pattern="[1-9][0-9]{0,19}"
          placeholder="全部操作者"
          :disabled="loading"
        />
      </div>
      <div v-if="scope === 'teams'">
        <label for="log-project">项目 ID</label
        ><input
          id="log-project"
          v-model="filters.project_id"
          inputmode="numeric"
          pattern="[1-9][0-9]{0,19}"
          placeholder="全部项目"
          :disabled="loading"
        />
      </div>
      <div>
        <label for="log-from">开始日期</label
        ><input
          id="log-from"
          v-model="filters.from"
          type="date"
          :disabled="loading"
        />
      </div>
      <div>
        <label for="log-to">结束日期</label
        ><input
          id="log-to"
          v-model="filters.to"
          type="date"
          :disabled="loading"
        />
      </div>
      <div class="activity-filter-actions">
        <el-button native-type="submit" type="primary" :disabled="loading"
          >查询日志</el-button
        ><el-button :disabled="loading" @click="reset">重置筛选</el-button
        ><el-button :loading="loading" @click="load">刷新日志</el-button>
      </div>
    </form>
    <p class="field-hint">
      日期按上海时区查询；操作者 ID 可在下方记录中查看，也可查询已离队成员。
    </p>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      class="alert"
    />
    <p v-if="loading" role="status" class="table-empty">正在查询操作日志…</p>
    <template v-if="result && !loading"
      ><p v-if="!result.items.length" class="table-empty">
        暂无符合条件的操作记录
      </p>
      <ol v-else class="activity-list">
        <li v-for="a in result.items" :key="a.id" :data-activity-id="a.id">
          <span class="activity-dot"></span>
          <div class="activity-entry">
            <strong>{{ a.actor_nickname }}</strong>
            {{ activityLabels[a.action] ?? a.action }}
            <p class="activity-object-name" v-if="text(a)">{{ text(a) }}</p>
            <p>
              {{ dateTime(a.created_at) }} · 操作者 #{{ a.actor_id }} ·
              {{ targetLabels[a.target_type] ?? a.target_type }} #{{
                a.target_id
              }}
            </p>
            <p v-if="transition(a)">{{ transition(a) }}</p>
            <div class="activity-links">
              <RouterLink v-if="a.project_id" :to="`/projects/${a.project_id}`"
                >查看项目 ↗</RouterLink
              ><RouterLink
                v-if="taskID(a) && a.action !== 'task.deleted'"
                :to="`/tasks/${taskID(a)}`"
                >查看任务 ↗</RouterLink
              >
            </div>
          </div>
        </li>
      </ol>
      <p class="field-hint" v-if="result.items.length">
        历史记录保留；关联资源已删除或已失去访问权时，详情会提示不可访问。
      </p>
      <PaginationBar
        :page="page"
        :total="result.total"
        :busy="loading"
        @change="changePage"
    /></template>
  </div>
</template>
