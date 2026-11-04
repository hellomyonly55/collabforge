<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from "vue";
import { taskAPI } from "../api/tasks";
import { APIError, errorMessage } from "../api/client";
import type { Project } from "../types/project";
import type { Task, TaskFields, TaskStatus } from "../types/task";
import { statusLabels, priorityLabels, statuses } from "../utils/task";
import { dateTime } from "../utils/team";
import CommentThread from "./CommentThread.vue";
import AssigneePicker from "./AssigneePicker.vue";
const props = defineProps<{ taskID?: string; project?: Project }>();
const emit = defineEmits<{
  created: [task: Task];
  changed: [];
  deleted: [projectID: string];
}>();
const task = ref<Task | null>(null),
  loading = ref(false),
  busy = ref(false),
  error = ref(""),
  notice = ref(""),
  moveStatus = ref<TaskStatus>("todo");
const form = reactive({
  title: "",
  description: "",
  priority: "medium" as TaskFields["priority"],
  assignee: "",
  due: "",
});
const teamID = computed(
  () => task.value?.team_id ?? props.project?.team_id ?? "",
);
const readonly = computed(() =>
  task.value
    ? !!(task.value.team_archived_at || task.value.project_archived_at)
    : !!(props.project?.team_archived_at || props.project?.archived_at),
);
const fields = (): TaskFields => ({
  title: form.title,
  description: form.description,
  priority: form.priority,
  assignee_id: form.assignee || null,
  due_at: form.due || null,
});
const hasEdits = computed(
  () =>
    !!task.value &&
    JSON.stringify(fields()) !==
      JSON.stringify({
        title: task.value.title,
        description: task.value.description,
        priority: task.value.priority,
        assignee_id: task.value.assignee_id,
        due_at: task.value.due_at,
      }),
);
let generation = 0;
onBeforeUnmount(() => generation++);
function setTask(value: Task) {
  task.value = value;
  form.title = value.title;
  form.description = value.description;
  form.priority = value.priority;
  form.assignee = value.assignee_id ?? "";
  form.due = value.due_at ?? "";
  moveStatus.value = value.status;
}
async function load() {
  const current = ++generation;
  loading.value = true;
  try {
    if (props.taskID) {
      const value = (await taskAPI.get(props.taskID)).data;
      if (current === generation) setTask(value);
    } else {
      task.value = null;
      form.title = "";
      form.description = "";
      form.priority = "medium";
      form.assignee = "";
      form.due = "";
    }
  } catch (e) {
    if (current === generation) {
      task.value = null;
      error.value = errorMessage(e);
    }
  } finally {
    if (current === generation) loading.value = false;
  }
}
watch(
  [() => props.taskID, () => props.project?.id],
  () => {
    error.value = "";
    notice.value = "";
    task.value = null;
    void load();
  },
  { immediate: true },
);
async function failure(e: unknown) {
  error.value = errorMessage(e);
  if (e instanceof APIError && [403, 404, 409].includes(e.status)) {
    await load();
    emit("changed");
  }
}
async function save() {
  if (busy.value || loading.value || readonly.value) return;
  error.value = "";
  notice.value = "";
  if (
    [...form.title.trim()].length < 1 ||
    [...form.title.trim()].length > 200 ||
    [...form.description.trim()].length > 2000
  ) {
    error.value = "标题为 1 至 200 字符，描述最多 2000 字符";
    return;
  }
  busy.value = true;
  try {
    if (task.value) {
      const value = (
        await taskAPI.update(task.value.id, task.value.version, fields())
      ).data;
      setTask(value);
      notice.value = "任务资料已保存";
      emit("changed");
    } else if (props.project) {
      const value = (await taskAPI.create(props.project.id, fields())).data;
      emit("created", value);
    }
  } catch (e) {
    await failure(e);
  } finally {
    busy.value = false;
  }
}
async function move() {
  if (!task.value || hasEdits.value || busy.value || readonly.value) return;
  busy.value = true;
  error.value = "";
  notice.value = "";
  try {
    const value = (
      await taskAPI.move(
        task.value.id,
        task.value.version,
        moveStatus.value,
        null,
      )
    ).data;
    setTask(value);
    notice.value = "任务状态已更新";
    emit("changed");
  } catch (e) {
    await failure(e);
  } finally {
    busy.value = false;
  }
}
async function remove() {
  if (
    !task.value ||
    !task.value.can_delete ||
    busy.value ||
    hasEdits.value ||
    readonly.value
  )
    return;
  if (!window.confirm("确认删除此任务？任务将从看板和列表中移除。")) return;
  busy.value = true;
  error.value = "";
  try {
    const projectID = task.value.project_id;
    await taskAPI.delete(task.value.id, task.value.version);
    emit("deleted", projectID);
  } catch (e) {
    await failure(e);
  } finally {
    busy.value = false;
  }
}
</script>
<template>
  <div class="task-panel">
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      class="alert"
    /><el-alert
      v-if="notice"
      :title="notice"
      type="success"
      :closable="false"
      class="alert"
    />
    <p v-if="loading" role="status">正在加载任务…</p>
    <div v-if="!loading && taskID && !task">
      <el-button
        @click="
          error = '';
          load();
        "
        >重新加载任务</el-button
      >
    </div>
    <template v-if="!loading && (task || (project && !taskID))"
      ><div v-if="task" class="task-panel-summary">
        <el-tag>{{ statusLabels[task.status] }}</el-tag
        ><el-tag v-if="task.is_overdue" type="danger">已逾期</el-tag
        ><span>#{{ task.id }} · {{ task.project_name }}</span
        ><RouterLink
          :to="`/projects/${task.project_id}/board`"
          class="activity-project-link"
          >查看项目看板</RouterLink
        >
      </div>
      <el-alert
        v-if="readonly"
        title="团队或项目已归档，任务仅可查看"
        type="info"
        :closable="false"
        class="alert"
      />
      <form v-if="!readonly" class="auth-form" @submit.prevent="save">
        <label for="task-title">任务标题</label
        ><el-input
          id="task-title"
          v-model="form.title"
          :disabled="busy"
          required
          placeholder="需要完成什么？"
        /><label for="task-description">任务描述</label
        ><el-input
          id="task-description"
          v-model="form.description"
          type="textarea"
          :rows="5"
          :disabled="busy"
          placeholder="补充目标、交付内容或验收要求"
        />
        <div class="task-form-grid">
          <div>
            <label for="task-priority">优先级</label
            ><select
              id="task-priority"
              v-model="form.priority"
              :disabled="busy"
            >
              <option
                v-for="(label, key) in priorityLabels"
                :key="key"
                :value="key"
              >
                {{ label }}
              </option>
            </select>
          </div>
          <div>
            <label for="task-due">截止日期</label
            ><input
              id="task-due"
              v-model="form.due"
              type="date"
              min="1000-01-01"
              max="9999-12-31"
              :disabled="busy"
            />
          </div>
        </div>
        <label for="task-assignee">负责人</label
        ><AssigneePicker
          id="task-assignee"
          v-model="form.assignee"
          :team-i-d="teamID"
          :current-name="task?.assignee_nickname"
          :disabled="busy"
        />
        <p v-if="!task" class="field-hint">
          新任务将进入待办列，可暂不分配负责人。
        </p>
        <el-button type="primary" native-type="submit" :loading="busy">{{
          task ? "保存任务资料" : "确认创建任务"
        }}</el-button
        ><el-button
          v-if="task && hasEdits"
          :disabled="busy"
          @click="setTask(task)"
          >重置修改</el-button
        >
      </form>
      <div v-else-if="task" class="task-readonly">
        <h2>{{ task.title }}</h2>
        <p class="task-description-text">
          {{ task.description || "暂无任务描述" }}
        </p>
        <p>优先级：{{ priorityLabels[task.priority] }}</p>
        <p>负责人：{{ task.assignee_nickname || "未分配" }}</p>
        <p>截止日期：{{ task.due_at || "未设置" }}</p>
      </div>
      <div v-if="task && !readonly" class="task-state-section">
        <h3>调整任务状态</h3>
        <p v-if="hasEdits" class="field-hint">
          请先保存当前修改，再调整状态或删除任务。
        </p>
        <div class="task-state-controls">
          <label for="task-move-status" class="sr-only">移动到</label
          ><select
            id="task-move-status"
            v-model="moveStatus"
            :disabled="busy || hasEdits"
          >
            <option v-for="status in statuses" :key="status" :value="status">
              {{ statusLabels[status] }}
            </option></select
          ><el-button :disabled="busy || hasEdits" @click="move"
            >移到该列末尾</el-button
          >
        </div>
        <el-button
          v-if="task.can_delete"
          class="task-delete"
          type="danger"
          plain
          :disabled="busy || hasEdits"
          @click="remove"
          >删除任务</el-button
        >
      </div>
      <CommentThread
        v-if="task"
        :task-i-d="task.id"
        :readonly="readonly"
        @refresh-task="load"
      />
      <div v-if="task" class="task-panel-footer">
        <span>创建者：{{ task.creator_nickname }}</span
        ><span>创建于 {{ dateTime(task.created_at) }}</span
        ><span>更新于 {{ dateTime(task.updated_at) }}</span>
      </div></template
    >
  </div>
</template>
