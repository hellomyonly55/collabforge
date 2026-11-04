<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useRoute } from "vue-router";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import PaginationBar from "../../components/PaginationBar.vue";
import TaskPanel from "../../components/TaskPanel.vue";
import TaskFiltersPanel from "../../components/TaskFilters.vue";
import { taskAPI } from "../../api/tasks";
import { projectAPI } from "../../api/projects";
import { APIError, errorMessage } from "../../api/client";
import type { Project } from "../../types/project";
import type {
  Task,
  TaskFilters,
  TaskPage,
  BoardColumn,
  TaskStatus,
} from "../../types/task";
import {
  emptyTaskFilters,
  statusLabels,
  priorityLabels,
} from "../../utils/task";
const route = useRoute(),
  id = computed(() => String(route.params.projectId));
const project = ref<Project | null>(null),
  columns = ref<BoardColumn[]>([]),
  list = ref<TaskPage | null>(null),
  revision = ref(0),
  mode = ref<"board" | "list">("board"),
  page = ref(1),
  filters = ref(emptyTaskFilters()),
  applied = ref(emptyTaskFilters()),
  loading = ref(false),
  busy = ref(false),
  error = ref(""),
  notice = ref(""),
  drawer = ref(false),
  selected = ref<string | undefined>(),
  dragged = ref<Task | null>(null);
const canWrite = computed(
  () =>
    !!project.value &&
    !project.value.archived_at &&
    !project.value.team_archived_at,
);
let generation = 0;
onBeforeUnmount(() => generation++);
async function load() {
  const current = ++generation,
    projectID = id.value,
    filter = { ...applied.value };
  loading.value = true;
  error.value = "";
  columns.value = [];
  list.value = null;
  try {
    if (mode.value === "board") {
      const data = (await taskAPI.board(projectID, filter)).data;
      if (current !== generation) return;
      project.value = data.project;
      columns.value = data.columns;
      revision.value = data.revision;
    } else {
      const [p, data] = await Promise.all([
        projectAPI.get(projectID),
        taskAPI.list(projectID, filter, page.value),
      ]);
      if (current !== generation) return;
      project.value = p.data;
      list.value = data.data;
    }
  } catch (e) {
    if (current === generation) {
      project.value = null;
      drawer.value = false;
      error.value = errorMessage(e);
    }
  } finally {
    if (current === generation) loading.value = false;
  }
}
function apply() {
  page.value = 1;
  applied.value = { ...filters.value };
  notice.value = "";
  void load();
}
function reset() {
  filters.value = emptyTaskFilters();
  apply();
}
function switchMode(value: typeof mode.value) {
  if (busy.value || loading.value) return;
  mode.value = value;
  page.value = 1;
  void load();
}
function open(task?: Task) {
  if (busy.value || loading.value) return;
  selected.value = task?.id;
  drawer.value = true;
}
async function created() {
  drawer.value = false;
  notice.value = "任务已创建";
  await load();
}
async function deleted() {
  drawer.value = false;
  notice.value = "任务已删除";
  await load();
}
async function more(column: BoardColumn) {
  if (busy.value || loading.value) return;
  busy.value = true;
  error.value = "";
  try {
    const result = (
      await taskAPI.list(
        id.value,
        { ...applied.value, status: column.status },
        Math.floor(column.items.length / 50) + 1,
        50,
        revision.value,
      )
    ).data;
    column.items = [...column.items, ...result.items];
    column.total = result.total;
    column.has_more = column.items.length < column.total;
  } catch (e) {
    error.value = errorMessage(e);
    if (e instanceof APIError && e.status === 409) {
      notice.value = "看板已变化，已重新加载";
      await load();
    } else if (e instanceof APIError && [403, 404].includes(e.status)) {
      await load();
    }
  } finally {
    busy.value = false;
  }
}
function dragStart(task: Task, event: DragEvent) {
  if (!canWrite.value || busy.value || loading.value) {
    event.preventDefault();
    return;
  }
  dragged.value = task;
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", task.id);
  }
}
async function moveTask(task: Task, status: TaskStatus, before: string | null) {
  if (!canWrite.value || busy.value || loading.value || before === task.id)
    return;
  busy.value = true;
  error.value = "";
  notice.value = "";
  try {
    await taskAPI.move(task.id, task.version, status, before);
    notice.value = "任务位置已更新";
  } catch (e) {
    notice.value = errorMessage(e);
  } finally {
    dragged.value = null;
    await load();
    busy.value = false;
  }
}
async function drop(status: TaskStatus, before: string | null) {
  const task = dragged.value;
  dragged.value = null;
  if (task) await moveTask(task, status, before);
}
function earlier(column: BoardColumn, index: number) {
  if (index > 0)
    void moveTask(
      column.items[index]!,
      column.status,
      column.items[index - 1]!.id,
    );
}
function later(column: BoardColumn, index: number) {
  const task = column.items[index];
  if (!task || index + 1 >= column.items.length) return;
  const before = column.items[index + 2]?.id ?? null;
  if (before === null && column.has_more) return;
  void moveTask(task, column.status, before);
}
function changePage(value: number) {
  page.value = value;
  void load();
}
watch(
  id,
  () => {
    project.value = null;
    drawer.value = false;
    mode.value = "board";
    page.value = 1;
    filters.value = emptyTaskFilters();
    applied.value = emptyTaskFilters();
    notice.value = "";
    void load();
  },
  { immediate: true },
);
</script>
<template>
  <WorkspaceLayout
    ><RouterLink :to="`/projects/${id}`" class="back-link"
      >← 返回项目</RouterLink
    >
    <div class="team-page-heading">
      <div>
        <div class="eyebrow">TASK BOARD</div>
        <h1 class="workspace-title">任务看板</h1>
        <p v-if="project">{{ project.name }} · {{ project.team_name }}</p>
      </div>
      <div class="board-heading-actions">
        <RouterLink :to="`/projects/${id}/statistics`" class="statistics-entry"
          >数据统计 ↗</RouterLink
        >
        <div class="view-switch">
          <button
            :class="{ selected: mode === 'board' }"
            :disabled="busy || loading"
            @click="switchMode('board')"
          >
            看板</button
          ><button
            :class="{ selected: mode === 'list' }"
            :disabled="busy || loading"
            @click="switchMode('list')"
          >
            列表
          </button>
        </div>
        <el-button
          v-if="canWrite"
          type="primary"
          :disabled="busy || loading"
          @click="open()"
          >创建任务</el-button
        ><el-button :loading="loading" :disabled="busy" @click="load"
          >刷新任务</el-button
        >
      </div>
    </div>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      class="alert" /><el-alert
      v-if="notice"
      :title="notice"
      type="info"
      :closable="false"
      class="alert" /><el-alert
      v-if="project && !canWrite"
      title="团队或项目已归档，任务仅可查看"
      type="info"
      :closable="false"
      class="alert" /><TaskFiltersPanel
      v-if="project"
      v-model="filters"
      :team-i-d="project.team_id"
      :busy="busy || loading"
      @apply="apply"
      @reset="reset" />
    <p v-if="mode === 'board' && project" class="board-hint">
      拖拽到卡片上方可插入其前面，拖到列内空白处放到列末尾。也可使用前移、后移按钮；手机端可在任务详情调整状态。
    </p>
    <div v-if="loading" class="team-empty" role="status">正在加载任务…</div>
    <div v-else-if="project && mode === 'board'" class="kanban-scroll">
      <div class="kanban">
        <section
          v-for="column in columns"
          :key="column.status"
          class="kanban-column"
          :data-status="column.status"
          :aria-label="statusLabels[column.status] + '列'"
          @dragover.prevent
          @drop.prevent="drop(column.status, null)"
        >
          <div class="column-heading">
            <span :class="['column-dot', column.status]"></span>
            <h2>{{ statusLabels[column.status] }}</h2>
            <span class="column-count">{{ column.total }}</span>
          </div>
          <div class="column-body">
            <article
              v-for="(task, index) in column.items"
              :key="task.id"
              :data-task-id="task.id"
              :class="['task-card', { 'is-dragging': dragged?.id === task.id }]"
              :draggable="canWrite && !busy"
              @dragstart="dragStart(task, $event)"
              @dragend="dragged = null"
              @dragover.prevent
              @drop.stop.prevent="drop(column.status, task.id)"
            >
              <div class="task-card-top">
                <span :class="['task-priority', 'priority-' + task.priority]"
                  >{{ priorityLabels[task.priority] }}优先级</span
                ><span class="task-number">#{{ task.id }}</span>
              </div>
              <button class="task-title-button" @click="open(task)">
                {{ task.title }}
              </button>
              <p v-if="task.description" class="task-card-description">
                {{ task.description }}
              </p>
              <div class="task-card-bottom">
                <span class="task-assignee"
                  ><span class="task-avatar">{{
                    task.assignee_nickname?.slice(0, 1) || "–"
                  }}</span
                  >{{ task.assignee_nickname || "未分配" }}</span
                ><span
                  v-if="task.due_at"
                  :class="{ 'task-overdue': task.is_overdue }"
                  >{{ task.is_overdue ? "逾期 " : "" }}{{ task.due_at }}</span
                >
              </div>
              <div v-if="canWrite" class="task-order-actions">
                <button
                  :disabled="busy || index === 0"
                  :aria-label="'前移 ' + task.title"
                  @click="earlier(column, index)"
                >
                  前移</button
                ><button
                  :disabled="
                    busy ||
                    index + 1 >= column.items.length ||
                    (column.has_more && index + 2 >= column.items.length)
                  "
                  :aria-label="'后移 ' + task.title"
                  @click="later(column, index)"
                >
                  后移
                </button>
              </div>
            </article>
            <p v-if="!column.items.length" class="column-empty">暂无任务</p>
          </div>
          <el-button
            v-if="column.has_more"
            class="column-more"
            :disabled="busy"
            @click="more(column)"
            >加载更多（{{ column.items.length }} /
            {{ column.total }}）</el-button
          >
        </section>
      </div>
    </div>
    <div v-else-if="project && list" class="team-panel">
      <div class="table-scroll">
        <table class="team-table task-table">
          <thead>
            <tr>
              <th>任务</th>
              <th>状态</th>
              <th>优先级</th>
              <th>负责人</th>
              <th>截止日期</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="task in list.items" :key="task.id">
              <td>
                <button class="task-title-button" @click="open(task)">
                  {{ task.title }}
                </button>
              </td>
              <td>{{ statusLabels[task.status] }}</td>
              <td>{{ priorityLabels[task.priority] }}</td>
              <td>{{ task.assignee_nickname || "未分配" }}</td>
              <td :class="{ 'task-overdue': task.is_overdue }">
                {{ task.due_at || "未设置" }}
              </td>
            </tr>
            <tr v-if="!list.items.length">
              <td colspan="5" class="table-empty">暂无符合筛选条件的任务</td>
            </tr>
          </tbody>
        </table>
      </div>
      <PaginationBar
        :page="page"
        :total="list.total"
        :busy="busy || loading"
        @change="changePage"
      />
    </div>
    <el-drawer
      v-model="drawer"
      :title="selected ? '任务详情' : '创建任务'"
      size="min(580px, 100vw)"
      append-to-body
      destroy-on-close
      ><TaskPanel
        v-if="drawer && project"
        :key="selected ?? 'new'"
        :task-i-d="selected"
        :project="project"
        @created="created"
        @changed="load"
        @deleted="deleted" /></el-drawer
  ></WorkspaceLayout>
</template>
