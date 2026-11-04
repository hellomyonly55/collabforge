<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { useRoute } from "vue-router";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import PaginationBar from "../../components/PaginationBar.vue";
import { adminAPI } from "../../api/admin";
import { APIError, errorMessage } from "../../api/client";
import type { AdminUser, AdminTeam, AdminStatistics } from "../../types/admin";
import type { Page } from "../../types/team";
import { useAuthStore } from "../../stores/auth";
import { dateTime } from "../../utils/team";
const route = useRoute(),
  auth = useAuthStore(),
  tab = computed(() =>
    route.path.endsWith("/users")
      ? "users"
      : route.path.endsWith("/teams")
        ? "teams"
        : "statistics",
  ),
  stats = ref<AdminStatistics | null>(null),
  users = ref<Page<AdminUser> | null>(null),
  teams = ref<Page<AdminTeam> | null>(null),
  page = ref(1),
  keyword = ref(""),
  status = ref("all"),
  role = ref("all"),
  filters = ref({ keyword: "", status: "all", role: "all" }),
  loading = ref(false),
  busy = ref(false),
  error = ref(""),
  notice = ref(""),
  denied = ref(false),
  target = ref<{
    id: string;
    label: string;
    action: "active" | "disabled" | "archive";
  } | null>(null),
  reason = ref("");
let generation = 0,
  scope = 0;
onBeforeUnmount(() => {
  generation++;
  scope++;
});
async function load() {
  const current = ++generation;
  loading.value = true;
  error.value = "";
  stats.value = null;
  users.value = null;
  teams.value = null;
  try {
    if (tab.value === "statistics") {
      const data = (await adminAPI.statistics()).data;
      if (current === generation) stats.value = data;
    } else if (tab.value === "users") {
      const data = (
        await adminAPI.users(
          page.value,
          filters.value.keyword,
          filters.value.status,
          filters.value.role,
        )
      ).data;
      if (current === generation) users.value = data;
    } else {
      const data = (
        await adminAPI.teams(
          page.value,
          filters.value.keyword,
          filters.value.status,
        )
      ).data;
      if (current === generation) teams.value = data;
    }
  } catch (e) {
    if (current === generation) {
      error.value = errorMessage(e);
      if (e instanceof APIError && e.status === 403) {
        denied.value = true;
        target.value = null;
        void auth.refresh().catch(() => {});
      }
    }
  } finally {
    if (current === generation) loading.value = false;
  }
}
watch(
  tab,
  () => {
    scope++;
    target.value = null;
    busy.value = false;
    page.value = 1;
    keyword.value = "";
    status.value = "all";
    role.value = "all";
    filters.value = { keyword: "", status: "all", role: "all" };
    notice.value = "";
    void load();
  },
  { immediate: true },
);
function search() {
  filters.value = {
    keyword: keyword.value.trim(),
    status: status.value,
    role: role.value,
  };
  page.value = 1;
  notice.value = "";
  void load();
}
function reset() {
  keyword.value = "";
  status.value = "all";
  role.value = "all";
  search();
}
function changePage(p: number) {
  page.value = p;
  void load();
}
function choose(
  id: string,
  label: string,
  action: "active" | "disabled" | "archive",
) {
  target.value = { id, label, action };
  reason.value = "";
  error.value = "";
  notice.value = "";
}
async function submit() {
  const selected = target.value;
  if (!selected || busy.value || !reason.value.trim()) return;
  const current = scope;
  busy.value = true;
  error.value = "";
  try {
    if (selected.action === "archive")
      await adminAPI.archive(selected.id, reason.value);
    else await adminAPI.status(selected.id, selected.action, reason.value);
    if (current !== scope) return;
    target.value = null;
    notice.value =
      selected.action === "archive"
        ? "团队已归档"
        : selected.action === "disabled"
          ? "账号已禁用，原有会话已失效"
          : "账号已启用，需要重新登录";
    await load();
  } catch (e) {
    if (current === scope) {
      error.value = errorMessage(e);
      if (e instanceof APIError && [403, 404, 409].includes(e.status)) {
        const message = error.value;
        target.value = null;
        await load();
        error.value = message;
      }
    }
  } finally {
    if (current === scope) busy.value = false;
  }
}
</script>
<template>
  <WorkspaceLayout
    ><div class="eyebrow">PLATFORM ADMIN</div>
    <div class="statistics-heading">
      <div>
        <h1 class="workspace-title">管理员后台</h1>
        <p class="workspace-subtitle">平台概览、账号状态与团队归档。</p>
      </div>
      <el-button :loading="loading" :disabled="busy" @click="load"
        >刷新后台</el-button
      >
    </div>
    <nav class="team-tabs admin-tabs" aria-label="管理导航">
      <RouterLink to="/admin" :class="{ selected: tab === 'statistics' }"
        >平台统计</RouterLink
      ><RouterLink to="/admin/users" :class="{ selected: tab === 'users' }"
        >用户管理</RouterLink
      ><RouterLink to="/admin/teams" :class="{ selected: tab === 'teams' }"
        >团队查看</RouterLink
      >
    </nav>
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
    <p v-if="loading" role="status">正在读取真实平台数据…</p>
    <template v-if="!denied">
      <template v-if="stats && !loading"
        ><div class="statistics-cards">
          <article class="statistic-card">
            <span>用户总数</span
            ><strong data-admin-stat="users">{{ stats.users }}</strong
            ><small
              >活跃 {{ stats.active_users }} · 禁用
              {{ stats.disabled_users }}</small
            >
          </article>
          <article class="statistic-card">
            <span>团队总数</span
            ><strong data-admin-stat="teams">{{ stats.teams }}</strong
            ><small>已归档 {{ stats.archived_teams }}</small>
          </article>
          <article class="statistic-card">
            <span>项目总数</span
            ><strong data-admin-stat="projects">{{ stats.projects }}</strong
            ><small>已归档 {{ stats.archived_projects }}</small>
          </article>
          <article class="statistic-card">
            <span>任务总数</span
            ><strong data-admin-stat="tasks">{{ stats.tasks }}</strong
            ><small
              >已完成 {{ stats.completed_tasks }} · 逾期
              {{ stats.overdue_tasks }}</small
            >
          </article>
        </div>
        <section class="team-panel admin-summary">
          <h2>平台运行概况</h2>
          <p>
            活跃平台管理员 <b>{{ stats.admins }}</b> 人 · 未删除评论
            <b>{{ stats.comments }}</b> 条
          </p>
          <p>
            任务与评论排除软删除，包含归档项目。逾期按 {{ stats.today }}（{{
              stats.timezone
            }}）判定。
          </p>
          <p>查询时间 {{ dateTime(stats.as_of) }}</p>
        </section></template
      >
      <section v-if="tab !== 'statistics'" class="team-panel">
        <form
          class="admin-filters"
          aria-label="后台筛选"
          @submit.prevent="search"
        >
          <div>
            <label for="admin-keyword">关键词</label
            ><input
              id="admin-keyword"
              v-model="keyword"
              maxlength="100"
              :placeholder="tab === 'users' ? '邮箱或昵称' : '团队名称或简介'"
              :disabled="busy || loading"
            />
          </div>
          <div>
            <label for="admin-status">状态</label
            ><select
              id="admin-status"
              v-model="status"
              :disabled="busy || loading"
            >
              <option value="all">全部状态</option>
              <option value="active">
                {{ tab === "users" ? "活跃" : "未归档" }}
              </option>
              <option :value="tab === 'users' ? 'disabled' : 'archived'">
                {{ tab === "users" ? "已禁用" : "已归档" }}
              </option>
            </select>
          </div>
          <div v-if="tab === 'users'">
            <label for="admin-role">平台角色</label
            ><select id="admin-role" v-model="role" :disabled="busy || loading">
              <option value="all">全部角色</option>
              <option value="user">普通用户</option>
              <option value="admin">平台管理员</option>
            </select>
          </div>
          <div class="admin-filter-actions">
            <el-button
              type="primary"
              native-type="submit"
              :disabled="busy || loading"
              >查询</el-button
            ><el-button :disabled="busy || loading" @click="reset"
              >重置</el-button
            >
          </div>
        </form>
        <p class="field-hint">
          {{
            tab === "users"
              ? "禁用不会转让团队所有权或清空任务分配；启用后需重新登录，旧会话不会恢复。"
              : "这里只查看团队基本资料与数量；平台角色不会获得团队任务编辑权限。"
          }}
        </p>
        <div v-if="users && !loading" class="table-scroll">
          <table class="team-table admin-table">
            <thead>
              <tr>
                <th>用户</th>
                <th>角色</th>
                <th>状态</th>
                <th>拥有团队</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="u in users.items" :key="u.id" :data-admin-user="u.id">
                <td>
                  <strong>{{ u.nickname }}</strong
                  ><small>{{ u.email }} · #{{ u.id }}</small>
                </td>
                <td>
                  {{ u.platform_role === "admin" ? "平台管理员" : "普通用户" }}
                </td>
                <td>{{ u.status === "active" ? "活跃" : "已禁用" }}</td>
                <td>{{ u.owned_team_count }}</td>
                <td>
                  <el-button
                    :type="u.status === 'active' ? 'danger' : 'primary'"
                    plain
                    :disabled="busy || u.id === auth.user?.id"
                    @click="
                      choose(
                        u.id,
                        u.email,
                        u.status === 'active' ? 'disabled' : 'active',
                      )
                    "
                    >{{
                      u.id === auth.user?.id
                        ? "当前账号"
                        : u.status === "active"
                          ? "禁用账号"
                          : "启用账号"
                    }}</el-button
                  >
                </td>
              </tr>
              <tr v-if="!users.items.length">
                <td colspan="5" class="table-empty">暂无符合条件的用户</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="teams && !loading" class="table-scroll">
          <table class="team-table admin-table">
            <thead>
              <tr>
                <th>团队</th>
                <th>所有者</th>
                <th>成员 / 项目 / 任务</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="t in teams.items" :key="t.id" :data-admin-team="t.id">
                <td>
                  <strong>{{ t.name }}</strong
                  ><small
                    >#{{ t.id }} · {{ t.description || "暂无简介" }}</small
                  >
                </td>
                <td>
                  {{ t.owner_nickname }}<small>{{ t.owner_email }}</small>
                </td>
                <td>
                  {{ t.member_count }} / {{ t.project_count }} /
                  {{ t.task_count }}
                </td>
                <td>{{ t.archived_at ? "已归档" : "未归档" }}</td>
                <td>
                  <el-button
                    type="danger"
                    plain
                    :disabled="busy || !!t.archived_at"
                    @click="choose(t.id, t.name, 'archive')"
                    >归档团队</el-button
                  >
                </td>
              </tr>
              <tr v-if="!teams.items.length">
                <td colspan="5" class="table-empty">暂无符合条件的团队</td>
              </tr>
            </tbody>
          </table>
        </div>
        <PaginationBar
          v-if="users || teams"
          :page="page"
          :total="users?.total ?? teams?.total ?? 0"
          :busy="busy || loading"
          @change="changePage"
        />
      </section>
      <section
        v-if="target"
        class="team-panel admin-confirm"
        aria-label="管理操作确认"
      >
        <h2>
          {{
            target.action === "archive"
              ? "归档团队"
              : target.action === "disabled"
                ? "禁用账号"
                : "启用账号"
          }}
        </h2>
        <p>{{ target.label }}</p>
        <p>
          {{
            target.action === "archive"
              ? "归档后整个团队及项目只读，当前不支持恢复。"
              : "操作将使该账号旧会话失效，不更改团队所有权。"
          }}
        </p>
        <form @submit.prevent="submit">
          <label for="admin-reason">操作原因</label
          ><el-input
            id="admin-reason"
            v-model="reason"
            type="textarea"
            maxlength="500"
            :disabled="busy"
            placeholder="填写原因，将记录到审计日志"
          />
          <div>
            <el-button
              type="danger"
              native-type="submit"
              :loading="busy"
              :disabled="!reason.trim()"
              >确认操作</el-button
            ><el-button :disabled="busy" @click="target = null">取消</el-button>
          </div>
        </form>
      </section>
    </template></WorkspaceLayout
  >
</template>
