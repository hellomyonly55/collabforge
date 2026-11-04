<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import PaginationBar from "../../components/PaginationBar.vue";
import { teamAPI } from "../../api/teams";
import { projectAPI } from "../../api/projects";
import { errorMessage } from "../../api/client";
import type { Page, Team } from "../../types/team";
import type { Project } from "../../types/project";
import { dateTime, roleLabels } from "../../utils/team";
const route = useRoute(),
  router = useRouter(),
  teamID = computed(() => String(route.params.teamId));
const team = ref<Team | null>(null),
  data = ref<Page<Project> | null>(null),
  page = ref(1),
  status = ref("all"),
  loading = ref(false),
  busy = ref(false),
  showCreate = ref(false),
  loadError = ref(""),
  actionError = ref("");
const form = reactive({ name: "", description: "" });
const canManage = computed(
  () =>
    !!team.value &&
    !team.value.archived_at &&
    ["owner", "admin"].includes(team.value.my_role),
);
let generation = 0;
onBeforeUnmount(() => generation++);
async function load(next = page.value) {
  const current = ++generation,
    id = teamID.value;
  page.value = next;
  loading.value = true;
  loadError.value = "";
  data.value = null;
  team.value = null;
  try {
    const [teamResponse, projects] = await Promise.all([
      teamAPI.get(id),
      projectAPI.list(id, next, status.value),
    ]);
    if (current !== generation) return;
    team.value = teamResponse.data;
    data.value = projects.data;
    if (!canManage.value) showCreate.value = false;
  } catch (e) {
    if (current === generation) loadError.value = errorMessage(e);
  } finally {
    if (current === generation) loading.value = false;
  }
}
async function create() {
  if (busy.value || loading.value) return;
  actionError.value = "";
  if (
    [...form.name.trim()].length < 1 ||
    [...form.name.trim()].length > 100 ||
    [...form.description.trim()].length > 2000
  ) {
    actionError.value = "项目名称为 1 至 100 字符，简介最多 2000 字符";
    return;
  }
  busy.value = true;
  try {
    const project = (
      await projectAPI.create(teamID.value, form.name, form.description)
    ).data;
    await router.push(`/projects/${project.id}`);
  } catch (e) {
    actionError.value = errorMessage(e);
    await load();
  } finally {
    busy.value = false;
  }
}
watch(
  teamID,
  () => {
    page.value = 1;
    status.value = "all";
    showCreate.value = false;
    actionError.value = "";
    form.name = "";
    form.description = "";
    void load(1);
  },
  { immediate: true },
);
</script>
<template>
  <WorkspaceLayout>
    <RouterLink :to="`/teams/${teamID}`" class="back-link"
      >← 返回团队</RouterLink
    >
    <div class="team-page-heading">
      <div>
        <div class="eyebrow">PROJECTS</div>
        <h1 class="workspace-title">团队项目</h1>
        <p v-if="team">
          {{ team.name }} · 我的角色：{{ roleLabels[team.my_role] }}
        </p>
      </div>
      <el-button
        v-if="canManage"
        type="primary"
        :disabled="busy || loading"
        @click="
          showCreate = !showCreate;
          actionError = '';
        "
        >创建项目</el-button
      >
    </div>
    <el-alert
      v-if="team?.archived_at"
      title="团队已归档，项目仅可查看"
      type="info"
      :closable="false"
      class="alert"
    />
    <el-alert
      v-if="loadError"
      :title="loadError"
      type="error"
      :closable="false"
      class="alert"
    />
    <el-alert
      v-if="actionError"
      :title="actionError"
      type="error"
      :closable="false"
      class="alert"
    />
    <section v-if="showCreate && canManage" class="team-panel team-form-panel">
      <h2>创建项目</h2>
      <form class="auth-form" @submit.prevent="create">
        <label for="project-name">项目名称</label
        ><el-input
          id="project-name"
          v-model="form.name"
          required
          :disabled="busy"
          placeholder="例如：CollabForge 协作平台"
        /><label for="project-description">项目简介</label
        ><el-input
          id="project-description"
          v-model="form.description"
          type="textarea"
          :rows="3"
          :disabled="busy"
          placeholder="描述项目目标与计划"
        />
        <p class="field-hint">团队内所有成员都可以查看项目。</p>
        <div>
          <el-button type="primary" native-type="submit" :loading="busy"
            >确认创建项目</el-button
          ><el-button :disabled="busy" @click="showCreate = false"
            >取消</el-button
          >
        </div>
      </form>
    </section>
    <div class="team-list-toolbar">
      <span>项目列表</span>
      <div>
        <label for="project-status" class="sr-only">项目状态</label
        ><select
          id="project-status"
          v-model="status"
          :disabled="loading || busy"
          @change="load(1)"
        >
          <option value="all">全部项目</option>
          <option value="active">进行中</option>
          <option value="archived">已归档</option></select
        ><el-button :loading="loading" :disabled="busy" @click="load()"
          >刷新</el-button
        >
      </div>
    </div>
    <div v-if="loading" class="team-empty" role="status">正在加载项目…</div>
    <div v-else-if="data?.items.length" class="team-cards">
      <article
        v-for="project in data.items"
        :key="project.id"
        class="team-card"
      >
        <div class="team-card-top">
          <span class="project-symbol">▧</span
          ><el-tag
            :type="
              project.archived_at || project.team_archived_at
                ? 'info'
                : 'success'
            "
            effect="plain"
            >{{
              project.team_archived_at
                ? "团队已归档"
                : project.archived_at
                  ? "已归档"
                  : "进行中"
            }}</el-tag
          >
        </div>
        <h2>{{ project.name }}</h2>
        <p class="team-description">
          {{ project.description || "暂无项目简介" }}
        </p>
        <div class="project-meta">
          <span>由 {{ project.creator_nickname }} 创建</span
          ><span>{{ dateTime(project.created_at) }}</span>
        </div>
        <RouterLink
          :to="`/projects/${project.id}`"
          :aria-label="`进入项目 ${project.name}`"
          class="team-enter"
          >进入项目 <span>↗</span></RouterLink
        >
      </article>
    </div>
    <div v-else-if="data" class="team-empty">
      <span class="welcome-symbol">▧</span>
      <h2>暂无项目</h2>
      <p>
        {{
          canManage
            ? "创建第一个项目，把团队目标逐步落地。"
            : "当前筛选条件下暂无项目，可切换状态或联系团队管理员。"
        }}
      </p>
    </div>
    <PaginationBar
      v-if="data"
      :page="page"
      :total="data.total"
      :busy="loading || busy"
      @change="load"
    />
  </WorkspaceLayout>
</template>
