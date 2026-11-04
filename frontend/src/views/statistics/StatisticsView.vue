<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useRoute } from "vue-router";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import { statisticsAPI } from "../../api/statistics";
import type { Statistics } from "../../types/statistics";
import { errorMessage } from "../../api/client";
import { statusLabels, priorityLabels } from "../../utils/task";
import { dateTime } from "../../utils/team";
const route = useRoute(),
  scope = computed((): "teams" | "projects" =>
    route.params.teamId ? "teams" : "projects",
  ),
  id = computed(() => String(route.params.teamId ?? route.params.projectId)),
  data = ref<Statistics | null>(null),
  loading = ref(false),
  error = ref("");
let generation = 0;
onBeforeUnmount(() => generation++);
async function load() {
  const current = ++generation;
  loading.value = true;
  error.value = "";
  data.value = null;
  try {
    const result = (await statisticsAPI.get(scope.value, id.value)).data;
    if (current === generation) data.value = result;
  } catch (e) {
    if (current === generation) error.value = errorMessage(e);
  } finally {
    if (current === generation) loading.value = false;
  }
}
watch([scope, id], load, { immediate: true });
const percent = (count: number, total: number) =>
  total ? Math.round((count / total) * 1000) / 10 : 0;
</script>
<template>
  <WorkspaceLayout>
    <RouterLink :to="`/${scope}/${id}`" class="back-link"
      >← 返回{{ scope === "teams" ? "团队" : "项目" }}</RouterLink
    >
    <div class="eyebrow">TEAM INSIGHTS</div>
    <div class="statistics-heading">
      <div>
        <h1 class="workspace-title">
          {{ scope === "teams" ? "团队" : "项目" }}统计
        </h1>
        <p class="workspace-subtitle">
          {{ data?.name ?? "根据真实任务，了解协作进展。" }}
        </p>
      </div>
      <el-button type="primary" :loading="loading" @click="load"
        >刷新统计</el-button
      >
    </div>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      class="alert"
    />
    <p v-if="loading" role="status" class="table-empty">
      正在查询真实任务统计…
    </p>
    <template v-if="data && !loading">
      <el-alert
        v-if="data.archived"
        title="团队或项目已归档，以下为保留任务的真实统计"
        type="info"
        :closable="false"
        class="alert"
      />
      <p class="statistics-context">
        排除已删除任务，包含已归档项目。逾期：未完成且截止日期早于
        {{ data.today }}（{{ data.timezone }}）。
      </p>
      <div class="statistics-cards">
        <article class="statistic-card">
          <span>任务总数</span
          ><strong data-stat="total">{{ data.total }}</strong
          ><small>当前范围内全部任务</small>
        </article>
        <article class="statistic-card">
          <span>已完成</span
          ><strong data-stat="completed">{{ data.completed }}</strong
          ><small>完成率 {{ data.completion_rate }}%</small>
        </article>
        <article class="statistic-card">
          <span>未完成</span
          ><strong data-stat="uncompleted">{{ data.uncompleted }}</strong
          ><small>待办、进行中与待验收</small>
        </article>
        <article class="statistic-card statistic-overdue">
          <span>已逾期</span
          ><strong data-stat="overdue">{{ data.overdue }}</strong
          ><small>已完成任务不计入逾期</small>
        </article>
      </div>
      <div v-if="data.total === 0" class="statistics-empty" role="status">
        当前范围还没有任务，统计均为零。创建真实任务后刷新即可查看变化。
      </div>
      <div class="statistics-charts">
        <section class="statistics-chart team-panel" aria-label="按状态统计">
          <h2>任务状态分布</h2>
          <div class="statistics-bars">
            <div
              v-for="row in data.by_status"
              :key="row.key"
              class="statistics-bar-row"
              :data-status-stat="row.key"
            >
              <span>{{
                statusLabels[row.key as keyof typeof statusLabels]
              }}</span>
              <div class="statistics-track" aria-hidden="true">
                <i
                  :class="`statistics-${row.key}`"
                  :style="{ width: `${percent(row.count, data.total)}%` }"
                ></i>
              </div>
              <strong>{{ row.count }}</strong
              ><small>{{ percent(row.count, data.total) }}%</small>
            </div>
          </div>
          <div class="completion-summary">
            <div
              class="completion-ring"
              :style="{
                background: `conic-gradient(#5b6ef5 ${data.completion_rate}%, #edf1f9 0)`,
              }"
              role="img"
              :aria-label="`完成率 ${data.completion_rate}%`"
            >
              <span>{{ data.completion_rate }}%</span>
            </div>
            <div>
              <b>完成率</b>
              <p>{{ data.completed }} / {{ data.total }} 项任务已完成</p>
            </div>
          </div>
        </section>
        <section class="statistics-chart team-panel" aria-label="按优先级统计">
          <h2>任务优先级</h2>
          <div class="statistics-bars">
            <div
              v-for="row in data.by_priority"
              :key="row.key"
              class="statistics-bar-row"
              :data-priority-stat="row.key"
            >
              <span>{{
                priorityLabels[row.key as keyof typeof priorityLabels]
              }}</span>
              <div class="statistics-track" aria-hidden="true">
                <i
                  :class="`statistics-${row.key}`"
                  :style="{ width: `${percent(row.count, data.total)}%` }"
                ></i>
              </div>
              <strong>{{ row.count }}</strong
              ><small>{{ percent(row.count, data.total) }}%</small>
            </div>
          </div>
          <p class="statistics-chart-note">
            按当前优先级汇总全部未删除任务，包含已完成任务。
          </p>
        </section>
      </div>
      <section
        class="team-panel statistics-assignees"
        aria-label="按负责人统计"
      >
        <h2>负责人任务分布</h2>
        <p>显示有任务的负责人及未分配任务；人数不等于当前团队成员总数。</p>
        <div class="table-scroll">
          <table class="team-table statistics-table">
            <thead>
              <tr>
                <th>负责人</th>
                <th>任务数</th>
                <th>已完成</th>
                <th>未完成</th>
                <th>已逾期</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in data.by_assignee"
                :key="row.user_id ?? 'unassigned'"
                :data-assignee-stat="row.user_id ?? 'unassigned'"
              >
                <td>
                  <strong>{{ row.nickname }}</strong
                  ><small v-if="row.user_id">#{{ row.user_id }}</small>
                </td>
                <td>{{ row.total }}</td>
                <td>{{ row.completed }}</td>
                <td>{{ row.total - row.completed }}</td>
                <td :class="{ 'task-overdue': row.overdue > 0 }">
                  {{ row.overdue }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
      <p class="statistics-updated">
        查询时间 {{ dateTime(data.as_of) }} ·
        其他成员更新后点击“刷新统计”读取最新结果。
      </p>
    </template>
  </WorkspaceLayout>
</template>
