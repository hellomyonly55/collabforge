<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import ActivityFeed from "../../components/ActivityFeed.vue";
import { projectAPI } from "../../api/projects";
import { errorMessage } from "../../api/client";
import type { Project } from "../../types/project";
import { dateTime, roleLabels } from "../../utils/team";
const route = useRoute(),
  id = computed(() => String(route.params.projectId)),
  project = ref<Project | null>(null),
  tab = ref<"overview" | "activities">("overview"),
  loading = ref(false),
  busy = ref(false),
  loadError = ref(""),
  actionError = ref(""),
  notice = ref("");
const form = reactive({ name: "", description: "" });
const canManage = computed(
  () =>
    !!project.value &&
    !project.value.archived_at &&
    !project.value.team_archived_at &&
    ["owner", "admin"].includes(project.value.my_role),
);
let generation = 0;
onBeforeUnmount(() => generation++);
async function load() {
  const current = ++generation,
    projectID = id.value;
  loading.value = true;
  loadError.value = "";

  try {
    const data = (await projectAPI.get(projectID)).data;
    if (current !== generation) return;
    project.value = data;
    form.name = data.name;
    form.description = data.description;
  } catch (e) {
    if (current === generation) {
      project.value = null;
      loadError.value = errorMessage(e);
    }
  } finally {
    if (current === generation) loading.value = false;
  }
}
function switchTab(value: typeof tab.value) {
  if (busy.value || loading.value) return;
  tab.value = value;
  actionError.value = "";
  notice.value = "";
  void load();
}
async function perform(action: () => Promise<unknown>, success: string) {
  if (busy.value || loading.value) return;
  busy.value = true;
  actionError.value = "";
  notice.value = "";
  try {
    await action();
    notice.value = success;
    await load();
  } catch (e) {
    actionError.value = errorMessage(e);
    await load();
  } finally {
    busy.value = false;
  }
}
function save() {
  if (
    [...form.name.trim()].length < 1 ||
    [...form.name.trim()].length > 100 ||
    [...form.description.trim()].length > 2000
  ) {
    actionError.value = "项目名称为 1 至 100 字符，简介最多 2000 字符";
    return;
  }
  void perform(
    () => projectAPI.update(id.value, form.name, form.description),
    "项目资料已保存",
  );
}
function archive() {
  if (window.confirm("确认归档项目？归档后仅可查看，当前版本不支持恢复。"))
    void perform(() => projectAPI.archive(id.value), "项目已归档");
}
watch(
  id,
  () => {
    project.value = null;

    tab.value = "overview";
    actionError.value = "";
    notice.value = "";
    void load();
  },
  { immediate: true },
);
</script>
<template>
  <WorkspaceLayout>
    <RouterLink
      v-if="project"
      :to="`/teams/${project.team_id}/projects`"
      class="back-link"
      >← 返回团队项目</RouterLink
    ><RouterLink v-else to="/teams" class="back-link"
      >← 返回团队空间</RouterLink
    >
    <el-alert
      v-if="loadError"
      :title="loadError"
      type="error"
      :closable="false"
      class="alert"
    /><el-alert
      v-if="actionError"
      :title="actionError"
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
    <div v-if="!project && loading" class="team-empty" role="status">
      正在加载项目…
    </div>
    <div v-else-if="!project">
      <el-button :loading="loading" @click="load">重试</el-button>
    </div>
    <template v-if="project"
      ><div class="team-detail-heading">
        <div>
          <RouterLink
            :to="`/projects/${id}/statistics`"
            class="statistics-entry"
            >数据统计 ↗</RouterLink
          >
          <div class="eyebrow">PROJECT WORKSPACE</div>
          <h1 class="workspace-title">{{ project.name }}</h1>
          <p>
            <RouterLink :to="`/teams/${project.team_id}`">{{
              project.team_name
            }}</RouterLink>
            · 我的角色：{{ roleLabels[project.my_role] }}
          </p>
        </div>
        <el-tag
          :type="
            project.archived_at || project.team_archived_at ? 'info' : 'success'
          "
          >{{
            project.team_archived_at
              ? "团队已归档 · 仅可查看"
              : project.archived_at
                ? "项目已归档 · 仅可查看"
                : "项目进行中"
          }}</el-tag
        >
      </div>
      <nav class="team-tabs" aria-label="项目页面">
        <RouterLink :to="`/projects/${id}/board`" class="team-project-link"
          >任务看板</RouterLink
        >
        <button
          :class="{ selected: tab === 'overview' }"
          :disabled="busy || loading"
          @click="switchTab('overview')"
        >
          项目概览</button
        ><button
          :class="{ selected: tab === 'activities' }"
          :disabled="busy || loading"
          @click="switchTab('activities')"
        >
          项目动态
        </button>
      </nav>
      <div v-if="loading" class="team-loading" role="status">正在刷新…</div>
      <section v-if="tab === 'overview'" class="team-panel">
        <h2>项目信息</h2>
        <form
          v-if="canManage"
          class="auth-form team-edit-form"
          @submit.prevent="save"
        >
          <label for="edit-project-name">项目名称</label
          ><el-input
            id="edit-project-name"
            v-model="form.name"
            required
            :disabled="busy || loading"
          /><label for="edit-project-description">项目简介</label
          ><el-input
            id="edit-project-description"
            v-model="form.description"
            type="textarea"
            :rows="5"
            :disabled="busy || loading"
          /><el-button
            type="primary"
            native-type="submit"
            :loading="busy"
            :disabled="loading"
            >保存项目资料</el-button
          >
        </form>
        <p v-else class="team-overview-description">
          {{ project.description || "暂无项目简介" }}
        </p>
        <div class="project-facts">
          <span>创建者：{{ project.creator_nickname }}</span
          ><span>创建于 {{ dateTime(project.created_at) }}</span
          ><span>更新于 {{ dateTime(project.updated_at) }}</span
          ><span v-if="project.archived_at"
            >归档于 {{ dateTime(project.archived_at) }}</span
          >
        </div>
        <div v-if="canManage" class="team-danger-zone">
          <p>归档后保留项目信息和历史记录，当前版本不支持恢复。</p>
          <el-button
            type="danger"
            plain
            :disabled="busy || loading"
            @click="archive"
            >归档项目</el-button
          >
        </div>
      </section>
      <section v-if="tab === 'activities'" class="team-panel">
        <div class="comment-heading">
          <h2>项目动态</h2>
          <RouterLink
            :to="`/projects/${id}/activities`"
            class="activity-project-link"
            >完整日志查询 ↗</RouterLink
          >
        </div>
        <ActivityFeed scope="projects" :resource-i-d="id" @unavailable="load" />
      </section>
      <p class="project-next-note">
        团队内所有成员可以通过任务看板创建、分配和推进任务。归档项目仅可查看。
      </p>
    </template>
  </WorkspaceLayout>
</template>
