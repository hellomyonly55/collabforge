<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { teamAPI } from "../../api/teams";
import { errorMessage } from "../../api/client";
import { useAuthStore } from "../../stores/auth";
import type { Invite, Member, Page, Team } from "../../types/team";
import { dateTime, inviteLabels, roleLabels } from "../../utils/team";
import WorkspaceLayout from "../../layouts/WorkspaceLayout.vue";
import ActivityFeed from "../../components/ActivityFeed.vue";
import PaginationBar from "../../components/PaginationBar.vue";
const route = useRoute(),
  router = useRouter(),
  auth = useAuthStore(),
  id = computed(() => String(route.params.teamId));
const team = ref<Team | null>(null),
  members = ref<Page<Member> | null>(null),
  invites = ref<Page<Invite> | null>(null);
const tab = ref<"overview" | "members" | "invites" | "activities">("overview"),
  loading = ref(false),
  busy = ref(false),
  loadError = ref(""),
  actionError = ref(""),
  notice = ref(""),
  newCode = ref(""),
  newInviteID = ref(""),
  copyNotice = ref("");
const page = ref(1),
  form = reactive({ name: "", description: "" }),
  inviteForm = reactive({ days: 7, uses: 20 });
const isOwner = computed(() => team.value?.my_role === "owner"),
  isManager = computed(() =>
    ["owner", "admin"].includes(team.value?.my_role ?? ""),
  ),
  canWrite = computed(() => !!team.value && !team.value.archived_at);
// A generation counter prevents stale requests from a previous route/tab replacing current state.
let generation = 0;
onBeforeUnmount(() => {
  generation++;
  newCode.value = "";
});
async function load() {
  const current = ++generation,
    teamID = id.value,
    section = tab.value,
    pageNumber = page.value;
  loading.value = true;
  loadError.value = "";
  members.value = null;
  invites.value = null;

  try {
    const data = (await teamAPI.get(teamID)).data;
    if (current !== generation) return;
    team.value = data;
    form.name = data.name;
    form.description = data.description;
    if (!["owner", "admin"].includes(data.my_role)) {
      newCode.value = "";
      if (section === "invites") {
        tab.value = "overview";
        return await load();
      }
    }
    if (data.archived_at) newCode.value = "";
    if (section === "members") {
      const data = (await teamAPI.members(teamID, pageNumber)).data;
      if (current === generation) members.value = data;
    }
    if (section === "invites") {
      const data = (await teamAPI.invites(teamID, pageNumber)).data;
      if (current === generation) invites.value = data;
    }
  } catch (e) {
    if (current === generation) {
      loadError.value = errorMessage(e);
      team.value = null;
      newCode.value = "";
    }
  } finally {
    if (current === generation) loading.value = false;
  }
}
function switchTab(value: typeof tab.value) {
  if (loading.value || busy.value) return;
  tab.value = value;
  page.value = 1;
  actionError.value = "";
  notice.value = "";
  void load();
}
async function perform(
  action: () => Promise<unknown>,
  success: string,
  leave = false,
) {
  if (busy.value || loading.value) return;
  busy.value = true;
  actionError.value = "";
  notice.value = "";
  try {
    await action();
    notice.value = success;
    if (leave) {
      await router.replace("/teams");
    } else {
      await load();
    }
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
    [...form.name.trim()].length > 80 ||
    [...form.description.trim()].length > 2000
  ) {
    actionError.value = "团队名称为 1 至 80 字符，简介最多 2000 字符";
    return;
  }
  void perform(
    () => teamAPI.update(id.value, form.name, form.description),
    "团队资料已更新",
  );
}
function changeRole(member: Member) {
  const role = member.role === "admin" ? "member" : "admin";
  void perform(
    () => teamAPI.role(id.value, member.user_id, role),
    "成员角色已更新",
  );
}
function remove(member: Member) {
  if (window.confirm(`确认移除 ${member.nickname}？对方将失去团队访问权限。`))
    void perform(() => teamAPI.remove(id.value, member.user_id), "成员已移除");
}
function transfer(member: Member) {
  if (
    window.confirm(
      `确认将团队所有权转让给 ${member.nickname}？你将成为管理员。`,
    )
  )
    void perform(
      () => teamAPI.transfer(id.value, member.user_id),
      "所有权已转让，你现在是管理员",
    );
}
function leave() {
  if (window.confirm("确认退出团队？再次加入需要有效邀请码。"))
    void perform(() => teamAPI.leave(id.value), "已退出团队", true);
}
function archive() {
  if (
    window.confirm("确认归档团队？归档后所有成员仅可查看，当前版本不支持恢复。")
  )
    void perform(() => teamAPI.archive(id.value), "团队已归档");
}
function revoke(invite: Invite) {
  if (window.confirm("确认撤销此邀请码？新成员将无法再使用它加入。"))
    void perform(async () => {
      await teamAPI.revoke(id.value, invite.id);
      if (newInviteID.value === invite.id) {
        newCode.value = "";
        newInviteID.value = "";
      }
    }, "邀请码已撤销");
}
async function createInvite() {
  if (busy.value || loading.value) return;
  newCode.value = "";
  copyNotice.value = "";
  await perform(async () => {
    const data = (
      await teamAPI.createInvite(
        id.value,
        Number(inviteForm.days),
        Number(inviteForm.uses),
      )
    ).data;
    newCode.value = data.code;
    newInviteID.value = data.invite.id;
  }, "邀请码已创建，请及时复制保存");
}
async function copy() {
  try {
    await navigator.clipboard.writeText(newCode.value);
    copyNotice.value = "已复制";
  } catch {
    copyNotice.value = "自动复制失败，请手动选择邀请码复制";
  }
}
function changePage(value: number) {
  page.value = value;
  void load();
}
watch(
  id,
  () => {
    newCode.value = "";
    tab.value = "overview";
    page.value = 1;
    actionError.value = "";
    notice.value = "";
    team.value = null;
    void load();
  },
  { immediate: true },
);
</script>
<template>
  <WorkspaceLayout
    ><RouterLink to="/teams" class="back-link">← 返回团队空间</RouterLink
    ><el-alert
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
    <div v-if="!team && loading" class="team-empty" role="status">
      正在加载团队…
    </div>
    <div v-else-if="!team">
      <el-button :loading="loading" @click="load">重试</el-button>
    </div>
    <template v-if="team"
      ><div class="team-detail-heading">
        <div>
          <RouterLink :to="`/teams/${id}/statistics`" class="statistics-entry"
            >数据统计 ↗</RouterLink
          >
          <div class="eyebrow">TEAM WORKSPACE</div>
          <h1 class="workspace-title">{{ team.name }}</h1>
          <p>
            {{ team.member_count }} 位成员
            <span class="team-divider">·</span> 我的角色：{{
              roleLabels[team.my_role]
            }}
          </p>
        </div>
        <el-tag :type="team.archived_at ? 'info' : 'success'">{{
          team.archived_at ? "已归档 · 仅可查看" : "协作进行中"
        }}</el-tag>
      </div>
      <nav class="team-tabs" aria-label="团队页面">
        <RouterLink :to="`/teams/${id}/projects`" class="team-project-link"
          >项目</RouterLink
        >
        <button
          :class="{ selected: tab === 'overview' }"
          :disabled="loading || busy"
          @click="switchTab('overview')"
        >
          概览</button
        ><button
          :class="{ selected: tab === 'members' }"
          :disabled="loading || busy"
          @click="switchTab('members')"
        >
          成员</button
        ><button
          v-if="isManager"
          :class="{ selected: tab === 'invites' }"
          :disabled="loading || busy"
          @click="switchTab('invites')"
        >
          邀请码</button
        ><button
          :class="{ selected: tab === 'activities' }"
          :disabled="loading || busy"
          @click="switchTab('activities')"
        >
          团队动态
        </button>
      </nav>
      <div v-if="loading" class="team-loading" role="status">正在刷新…</div>
      <section v-if="tab === 'overview'" class="team-panel">
        <h2>团队资料</h2>
        <form
          v-if="isManager && canWrite"
          class="auth-form team-edit-form"
          @submit.prevent="save"
        >
          <label for="edit-team-name">团队名称</label
          ><el-input
            id="edit-team-name"
            v-model="form.name"
            required
            :disabled="busy || loading"
          /><label for="edit-team-description">团队简介</label
          ><el-input
            id="edit-team-description"
            v-model="form.description"
            type="textarea"
            :rows="4"
            :disabled="busy || loading"
          /><el-button
            type="primary"
            native-type="submit"
            :disabled="loading"
            :loading="busy"
            >保存团队资料</el-button
          >
        </form>
        <p v-else class="team-overview-description">
          {{ team.description || "暂无团队简介" }}
        </p>
        <div class="team-facts">
          <span>创建于 {{ dateTime(team.created_at) }}</span
          ><span v-if="team.archived_at"
            >归档于 {{ dateTime(team.archived_at) }}</span
          >
        </div>
        <div v-if="canWrite" class="team-danger-zone">
          <template v-if="isOwner"
            ><p>
              退出团队前需要在成员页转让所有权。归档后团队仅可查看，当前版本不支持恢复。
            </p>
            <el-button
              type="danger"
              plain
              :disabled="busy || loading"
              @click="archive"
              >归档团队</el-button
            ></template
          ><el-button
            v-else
            type="danger"
            plain
            :disabled="busy || loading"
            @click="leave"
            >退出团队</el-button
          >
        </div>
      </section>
      <section v-if="tab === 'members'" class="team-panel">
        <div class="panel-heading">
          <h2>团队成员</h2>
          <span>权限变更立即生效</span>
        </div>
        <div v-if="members" class="table-scroll">
          <table class="team-table">
            <thead>
              <tr>
                <th>成员</th>
                <th>角色</th>
                <th>加入时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="member in members.items" :key="member.user_id">
                <td>
                  <strong
                    >{{ member.nickname }}
                    <small v-if="member.user_id === auth.user?.id"
                      >（我）</small
                    ></strong
                  >
                  <p>{{ member.email }}</p>
                  <el-tag
                    v-if="member.status === 'disabled'"
                    type="danger"
                    size="small"
                    >账号已禁用</el-tag
                  >
                </td>
                <td>
                  <el-tag type="info">{{ roleLabels[member.role] }}</el-tag>
                </td>
                <td>{{ dateTime(member.joined_at) }}</td>
                <td>
                  <div v-if="canWrite" class="table-actions">
                    <el-button
                      v-if="isOwner && member.role !== 'owner'"
                      size="small"
                      :disabled="busy || loading"
                      @click="changeRole(member)"
                      >{{
                        member.role === "admin" ? "设为成员" : "设为管理员"
                      }}</el-button
                    ><el-button
                      v-if="
                        isOwner &&
                        member.role !== 'owner' &&
                        member.status === 'active'
                      "
                      size="small"
                      :disabled="busy || loading"
                      @click="transfer(member)"
                      >转让所有权</el-button
                    ><el-button
                      v-if="
                        member.user_id !== auth.user?.id &&
                        member.role !== 'owner' &&
                        (isOwner || (isManager && member.role === 'member'))
                      "
                      size="small"
                      type="danger"
                      plain
                      :disabled="busy || loading"
                      @click="remove(member)"
                      >移除</el-button
                    >
                  </div>
                  <span
                    v-if="
                      !canWrite ||
                      (!isOwner &&
                        (!isManager ||
                          member.role !== 'member' ||
                          member.user_id === auth.user?.id))
                    "
                    class="muted"
                    >—</span
                  >
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <PaginationBar
          v-if="members"
          :page="page"
          :total="members.total"
          :busy="busy || loading"
          @change="changePage"
        />
      </section>
      <section v-if="tab === 'invites' && isManager" class="team-panel">
        <h2>邀请伙伴加入</h2>
        <p>邀请码仅创建时显示一次，请及时复制；列表只保留有效期和使用记录。</p>
        <form
          v-if="canWrite"
          class="invite-form"
          @submit.prevent="createInvite"
        >
          <div>
            <label for="invite-days">有效天数</label
            ><input
              id="invite-days"
              v-model.number="inviteForm.days"
              type="number"
              min="1"
              max="30"
              required
              :disabled="busy || loading"
            />
          </div>
          <div>
            <label for="invite-uses">最多使用次数</label
            ><input
              id="invite-uses"
              v-model.number="inviteForm.uses"
              type="number"
              min="1"
              max="100"
              required
              :disabled="busy || loading"
            />
          </div>
          <el-button
            type="primary"
            native-type="submit"
            :disabled="loading"
            :loading="busy"
            >创建邀请码</el-button
          >
        </form>
        <div v-if="newCode" class="invite-code-box">
          <label for="new-invite-code">新邀请码（离开页面后不再显示）</label
          ><el-input
            id="new-invite-code"
            :model-value="newCode"
            readonly
            type="textarea"
            :rows="2"
          /><el-button @click="copy">复制邀请码</el-button
          ><span role="status">{{ copyNotice }}</span>
        </div>
        <div v-if="invites" class="table-scroll">
          <table class="team-table">
            <thead>
              <tr>
                <th>编号</th>
                <th>有效期至</th>
                <th>使用次数</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="invite in invites.items" :key="invite.id">
                <td>#{{ invite.id }}</td>
                <td>{{ dateTime(invite.expires_at) }}</td>
                <td>{{ invite.used_count }} / {{ invite.max_uses }}</td>
                <td>
                  <el-tag
                    :type="
                      invite.status === 'active' && !team.archived_at
                        ? 'success'
                        : 'info'
                    "
                    >{{
                      team.archived_at
                        ? "团队已归档"
                        : inviteLabels[invite.status]
                    }}</el-tag
                  >
                </td>
                <td>
                  <el-button
                    v-if="canWrite && invite.status === 'active'"
                    size="small"
                    type="danger"
                    plain
                    :disabled="busy || loading"
                    @click="revoke(invite)"
                    >撤销</el-button
                  ><span v-else class="muted">—</span>
                </td>
              </tr>
              <tr v-if="!invites.items.length">
                <td colspan="5" class="table-empty">暂无邀请码</td>
              </tr>
            </tbody>
          </table>
        </div>
        <PaginationBar
          v-if="invites"
          :page="page"
          :total="invites.total"
          :busy="busy || loading"
          @change="changePage"
        />
      </section>
      <section v-if="tab === 'activities'" class="team-panel">
        <div class="comment-heading">
          <h2>团队动态</h2>
          <RouterLink
            :to="`/teams/${id}/activities`"
            class="activity-project-link"
            >完整日志查询 ↗</RouterLink
          >
        </div>
        <ActivityFeed scope="teams" :resource-i-d="id" @unavailable="load" />
      </section>
    </template>
  </WorkspaceLayout>
</template>
