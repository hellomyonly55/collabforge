<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue";
import { commentAPI } from "../api/comments";
import { APIError, errorMessage } from "../api/client";
import type { Comment } from "../types/comment";
import type { Page } from "../types/team";
import { dateTime } from "../utils/team";
import PaginationBar from "./PaginationBar.vue";
const props = defineProps<{ taskID: string; readonly: boolean }>();
const emit = defineEmits<{ refreshTask: [] }>();
const result = ref<Page<Comment> | null>(null),
  page = ref(1),
  loading = ref(false),
  busy = ref(false),
  error = ref(""),
  notice = ref(""),
  content = ref(""),
  editing = ref<Comment | null>(null),
  editedContent = ref(""),
  unavailable = ref(false);
let scope = 0,
  requestGeneration = 0;
onBeforeUnmount(() => {
  scope++;
  requestGeneration++;
});
async function load() {
  const current = ++requestGeneration;
  loading.value = true;
  result.value = null;
  try {
    const data = (await commentAPI.list(props.taskID, page.value)).data;
    if (current !== requestGeneration) return;
    if (data.total > 0 && data.items.length === 0 && page.value > 1) {
      page.value = Math.ceil(data.total / 20);
      await load();
      return;
    }
    if (data.total === 0) page.value = 1;
    result.value = data;
    unavailable.value = false;
  } catch (e) {
    if (current === requestGeneration) {
      error.value = errorMessage(e);
      if (e instanceof APIError && [403, 404].includes(e.status)) {
        unavailable.value = true;
        emit("refreshTask");
      }
    }
  } finally {
    if (current === requestGeneration) loading.value = false;
  }
}
watch(
  () => props.taskID,
  () => {
    scope++;
    page.value = 1;
    busy.value = false;
    content.value = "";
    editing.value = null;
    error.value = "";
    notice.value = "";
    void load();
  },
  { immediate: true },
);
async function refresh() {
  error.value = "";
  notice.value = "";
  editing.value = null;
  await load();
}
async function perform(
  action: () => Promise<unknown>,
  message: string,
  kind: "create" | "update" | "delete",
) {
  if (busy.value || loading.value || props.readonly || unavailable.value)
    return;
  const current = scope;
  busy.value = true;
  error.value = "";
  notice.value = "";
  try {
    await action();
    if (current !== scope) return;
    notice.value = message;
    editing.value = null;
    if (kind === "create") {
      content.value = "";
      page.value = 1;
    }
    await load();
  } catch (e) {
    if (current !== scope) return;
    error.value = errorMessage(e);
    if (e instanceof APIError && [403, 404, 409].includes(e.status)) {
      editing.value = null;
      await load();
      if (e.code === "PROJECT_ARCHIVED" || e.code === "TEAM_ARCHIVED")
        emit("refreshTask");
    }
  } finally {
    if (current === scope) busy.value = false;
  }
}
function create() {
  if (!content.value.trim()) {
    error.value = "请输入评论内容";
    return;
  }
  void perform(
    () => commentAPI.create(props.taskID, content.value),
    "评论已发表",
    "create",
  );
}
function edit(c: Comment) {
  editing.value = c;
  editedContent.value = c.content;
  error.value = "";
  notice.value = "";
}
function save() {
  const c = editing.value;
  if (!c) return;
  if (!editedContent.value.trim()) {
    error.value = "请输入评论内容";
    return;
  }
  void perform(
    () => commentAPI.update(c.id, editedContent.value, c.version),
    "评论已更新",
    "update",
  );
}
function remove(c: Comment) {
  if (window.confirm("确认删除这条评论？"))
    void perform(
      () => commentAPI.remove(c.id, c.version),
      "评论已删除",
      "delete",
    );
}
function changePage(value: number) {
  page.value = value;
  editing.value = null;
  error.value = "";
  notice.value = "";
  void load();
}
</script>
<template>
  <section class="comment-thread" aria-label="任务评论">
    <div class="comment-heading">
      <h3>
        任务评论 <small v-if="result">{{ result.total }}</small>
      </h3>
      <el-button :disabled="busy || loading" @click="refresh"
        >刷新评论</el-button
      >
    </div>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      class="alert"
    />
    <el-alert
      v-if="notice"
      :title="notice"
      type="success"
      :closable="false"
      class="alert"
    />
    <form
      v-if="!readonly && !unavailable"
      class="comment-compose"
      @submit.prevent="create"
    >
      <label :for="`new-comment-${taskID}`">发表评论</label
      ><el-input
        :id="`new-comment-${taskID}`"
        v-model="content"
        type="textarea"
        :rows="3"
        maxlength="2000"
        :disabled="busy"
        placeholder="补充进展、讨论问题或留下反馈"
      />
      <div class="comment-compose-footer">
        <span>纯文本，最多 2000 字符</span
        ><el-button
          type="primary"
          native-type="submit"
          :loading="busy"
          :disabled="loading || !content.trim()"
          >发表评论</el-button
        >
      </div>
    </form>
    <p v-else-if="readonly" class="field-hint">
      团队或项目已归档，评论仅可查看。
    </p>
    <p v-if="loading" role="status" class="table-empty">正在加载评论…</p>
    <template v-if="result && !loading">
      <p v-if="!result.items.length" class="comment-empty">
        暂无评论，讨论会从这里开始。
      </p>
      <ol v-else class="comment-list">
        <li v-for="c in result.items" :key="c.id" :data-comment-id="c.id">
          <div class="comment-meta">
            <span class="user-avatar">{{ c.author_nickname.slice(0, 1) }}</span
            ><strong>{{ c.author_nickname }}</strong
            ><time :datetime="c.created_at">{{ dateTime(c.created_at) }}</time>
          </div>
          <form
            v-if="editing?.id === c.id"
            class="comment-edit"
            @submit.prevent="save"
          >
            <label :for="`edit-comment-${c.id}`">编辑评论</label
            ><el-input
              :id="`edit-comment-${c.id}`"
              v-model="editedContent"
              type="textarea"
              :rows="3"
              maxlength="2000"
              :disabled="busy"
            />
            <div>
              <el-button
                type="primary"
                native-type="submit"
                :loading="busy"
                :disabled="!editedContent.trim()"
                >保存评论</el-button
              ><el-button :disabled="busy" @click="editing = null"
                >取消编辑</el-button
              >
            </div>
          </form>
          <p v-else class="comment-content">{{ c.content }}</p>
          <div v-if="!readonly" class="comment-actions">
            <el-button
              v-if="c.can_edit && editing?.id !== c.id"
              text
              :disabled="busy || loading"
              @click="edit(c)"
              >编辑评论</el-button
            ><el-button
              v-if="c.can_delete"
              text
              type="danger"
              :disabled="busy || loading || !!editing"
              @click="remove(c)"
              >删除评论</el-button
            >
          </div>
          <small v-if="c.version > 1" class="comment-edited"
            >更新于 {{ dateTime(c.updated_at) }}</small
          >
        </li>
      </ol>
      <PaginationBar
        :page="page"
        :total="result.total"
        :busy="busy || loading"
        @change="changePage"
      />
    </template>
  </section>
</template>
