<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "../../stores/auth";
import { teamAPI } from "../../api/teams";
import { errorMessage } from "../../api/client";
import type { Page, Team } from "../../types/team";
import { roleLabels } from "../../utils/team";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import PaginationBar from "../../components/PaginationBar.vue";
const auth = useAuthStore(),
  router = useRouter(),
  loading = ref(false),
  busy = ref(false),
  error = ref(""),
  actionError = ref(""),
  mode = ref<"create" | "join" | "">(""),
  page = ref(1),
  status = ref("all");
const data = ref<Page<Team> | null>(null),
  form = reactive({ name: "", description: "", code: "" });
async function load(next = page.value) {
  loading.value = true;
  error.value = "";
  data.value = null;
  page.value = next;
  try {
    data.value = (await teamAPI.list(page.value, status.value)).data;
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    loading.value = false;
  }
}
function open(value: "create" | "join") {
  mode.value = mode.value === value ? "" : value;
  actionError.value = "";
}
async function submit() {
  if (busy.value) return;
  actionError.value = "";
  if (
    mode.value === "create" &&
    ([...form.name.trim()].length < 1 ||
      [...form.name.trim()].length > 80 ||
      [...form.description.trim()].length > 2000)
  ) {
    actionError.value = "团队名称为 1 至 80 字符，简介最多 2000 字符";
    return;
  }
  if (mode.value === "join" && !/^[0-9a-fA-F]{64}$/.test(form.code.trim())) {
    actionError.value = "请输入完整的 64 位邀请码";
    return;
  }
  busy.value = true;
  try {
    let id: string;
    if (mode.value === "create") {
      id = (await teamAPI.create(form.name, form.description)).data.id;
    } else {
      id = (await teamAPI.join(form.code.trim())).data.team.id;
    }
    form.code = "";
    await router.push(`/teams/${id}`);
  } catch (e) {
    actionError.value = errorMessage(e);
  } finally {
    busy.value = false;
  }
}
onMounted(() => load());
</script>
<template>
  <WorkspaceLayout
    ><div class="eyebrow">YOUR WORKSPACE</div>
    <div class="team-page-heading">
      <div>
        <h1 class="workspace-title">团队空间</h1>
        <p>你好，{{ auth.user?.nickname }}。</p>
      </div>
      <div class="team-heading-actions">
        <el-button :disabled="busy" @click="open('join')">邀请码加入</el-button
        ><el-button type="primary" :disabled="busy" @click="open('create')"
          >创建团队</el-button
        >
      </div>
    </div>
    <section v-if="mode" class="team-panel team-form-panel">
      <h2>{{ mode === "create" ? "创建团队" : "加入团队" }}</h2>
      <el-alert
        v-if="actionError"
        :title="actionError"
        type="error"
        :closable="false"
        class="alert"
      />
      <form class="auth-form" @submit.prevent="submit">
        <template v-if="mode === 'create'"
          ><label for="team-name">团队名称</label
          ><el-input
            id="team-name"
            v-model="form.name"
            required
            :disabled="busy"
            placeholder="例如：CollabForge 开发小组" /><label
            for="team-description"
            >团队简介</label
          ><el-input
            id="team-description"
            v-model="form.description"
            type="textarea"
            :rows="3"
            :disabled="busy"
            placeholder="介绍团队的目标与分工" /></template
        ><template v-else
          ><label for="join-code">邀请码</label
          ><el-input
            id="join-code"
            v-model="form.code"
            type="textarea"
            :rows="2"
            :disabled="busy"
            required
            placeholder="粘贴团队管理员提供的完整邀请码"
          />
          <p class="field-hint">加入后默认为普通成员。</p></template
        >
        <div>
          <el-button type="primary" native-type="submit" :loading="busy">{{
            mode === "create" ? "确认创建" : "确认加入"
          }}</el-button
          ><el-button :disabled="busy" @click="mode = ''">取消</el-button>
        </div>
      </form>
    </section>
    <div class="team-list-toolbar">
      <span>我的团队</span>
      <div>
        <label for="team-status" class="sr-only">团队状态</label
        ><select
          id="team-status"
          v-model="status"
          :disabled="loading"
          @change="load(1)"
        >
          <option value="all">全部状态</option>
          <option value="active">进行中</option>
          <option value="archived">已归档</option></select
        ><el-button :loading="loading" @click="load()">刷新</el-button>
      </div>
    </div>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      class="alert"
    />
    <div v-if="loading" class="team-empty" role="status">正在加载团队…</div>
    <div v-else-if="data && data.items.length" class="team-cards">
      <article v-for="team in data.items" :key="team.id" class="team-card">
        <div class="team-card-top">
          <span class="team-symbol">{{ team.name.slice(0, 1) }}</span
          ><el-tag
            :type="team.archived_at ? 'info' : 'success'"
            effect="plain"
            >{{ team.archived_at ? "已归档" : "进行中" }}</el-tag
          >
        </div>
        <h2>{{ team.name }}</h2>
        <p class="team-description">{{ team.description || "暂无团队简介" }}</p>
        <div class="team-card-meta">
          <span>{{ team.member_count }} 位成员</span
          ><el-tag type="info" size="small">{{
            roleLabels[team.my_role]
          }}</el-tag>
        </div>
        <RouterLink
          :to="`/teams/${team.id}`"
          :aria-label="`进入团队 ${team.name}`"
          class="team-enter"
          >进入团队 <span>↗</span></RouterLink
        >
      </article>
    </div>
    <div v-else-if="data" class="team-empty">
      <span class="welcome-symbol">◈</span>
      <h2>{{ data.total ? "这一页暂无团队" : "这里还没有团队" }}</h2>
      <p>创建自己的团队，或使用邀请码加入伙伴的团队。</p>
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
