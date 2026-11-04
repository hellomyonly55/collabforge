<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue";
import { teamAPI } from "../api/teams";
import { errorMessage } from "../api/client";
import type { Member } from "../types/team";
const props = defineProps<{
  modelValue: string;
  teamID: string;
  id: string;
  disabled?: boolean;
  filter?: boolean;
  currentName?: string | null;
}>();
const emit = defineEmits<{ "update:modelValue": [value: string] }>();
const members = ref<Member[]>([]),
  page = ref(0),
  total = ref(0),
  loading = ref(false),
  error = ref("");
let generation = 0;
onBeforeUnmount(() => generation++);
async function load(next: number) {
  const current = ++generation;
  loading.value = true;
  error.value = "";
  try {
    const result = (await teamAPI.members(props.teamID, next, 100)).data;
    if (current !== generation) return;
    members.value =
      next === 1 ? result.items : [...members.value, ...result.items];
    page.value = next;
    total.value = result.total;
  } catch (e) {
    if (current === generation) error.value = errorMessage(e);
  } finally {
    if (current === generation) loading.value = false;
  }
}
watch(
  () => props.teamID,
  () => {
    members.value = [];
    page.value = 0;
    total.value = 0;
    if (props.teamID) void load(1);
  },
  { immediate: true },
);
</script>
<template>
  <div class="assignee-picker">
    <select
      :id="id"
      :value="modelValue"
      :disabled="disabled || loading"
      @change="
        emit('update:modelValue', ($event.target as HTMLSelectElement).value)
      "
    >
      <option value="">{{ filter ? "全部负责人" : "未分配" }}</option>
      <option v-if="filter" value="unassigned">未分配</option>
      <option
        v-if="
          modelValue &&
          modelValue !== 'unassigned' &&
          !members.some((m) => m.user_id === modelValue)
        "
        :value="modelValue"
      >
        {{ currentName || "当前负责人" }}
      </option>
      <option
        v-for="member in members"
        :key="member.user_id"
        :value="member.user_id"
        :disabled="!filter && member.status !== 'active'"
      >
        {{ member.nickname }} #{{ member.user_id
        }}{{ member.status === "disabled" ? "（已禁用）" : "" }}
      </option>
    </select>
    <p v-if="loading" class="field-hint">正在加载成员…</p>
    <p v-if="error" class="task-inline-error">
      {{ error }} <button type="button" @click="load(page + 1)">重试</button>
    </p>
    <button
      v-if="!loading && members.length < total"
      class="picker-more"
      type="button"
      :disabled="disabled"
      @click="load(page + 1)"
    >
      加载更多成员（{{ members.length }} / {{ total }}）
    </button>
  </div>
</template>
