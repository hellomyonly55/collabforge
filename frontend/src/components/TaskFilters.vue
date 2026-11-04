<script setup lang="ts">
import type { TaskFilters } from "../types/task";
import { statuses, statusLabels, priorityLabels } from "../utils/task";
import AssigneePicker from "./AssigneePicker.vue";
const props = defineProps<{
  modelValue: TaskFilters;
  teamID?: string;
  busy?: boolean;
  mine?: boolean;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: TaskFilters];
  apply: [];
  reset: [];
}>();
function update(key: keyof TaskFilters, value: string) {
  emit("update:modelValue", { ...props.modelValue, [key]: value });
}
</script>
<template>
  <form class="task-filters" @submit.prevent="emit('apply')">
    <div class="filter-keyword">
      <label for="task-keyword">关键词</label
      ><el-input
        id="task-keyword"
        :model-value="modelValue.keyword"
        :disabled="busy"
        placeholder="搜索标题或描述"
        @update:model-value="update('keyword', String($event))"
      />
    </div>
    <div>
      <label for="filter-task-status">状态</label
      ><select
        id="filter-task-status"
        :value="modelValue.status"
        :disabled="busy"
        @change="update('status', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">全部状态</option>
        <option v-for="status in statuses" :key="status" :value="status">
          {{ statusLabels[status] }}
        </option>
      </select>
    </div>
    <div>
      <label for="filter-task-priority">优先级</label
      ><select
        id="filter-task-priority"
        :value="modelValue.priority"
        :disabled="busy"
        @change="update('priority', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">全部优先级</option>
        <option v-for="(label, key) in priorityLabels" :key="key" :value="key">
          {{ label }}
        </option>
      </select>
    </div>
    <div v-if="teamID && !mine">
      <label for="filter-task-assignee">负责人</label
      ><AssigneePicker
        id="filter-task-assignee"
        :model-value="modelValue.assignee_id"
        :team-i-d="teamID"
        :disabled="busy"
        filter
        @update:model-value="update('assignee_id', $event)"
      />
    </div>
    <div>
      <label for="filter-task-overdue">截止情况</label
      ><select
        id="filter-task-overdue"
        :value="modelValue.overdue"
        :disabled="busy"
        @change="update('overdue', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">全部</option>
        <option value="true">已逾期</option>
        <option value="false">未逾期</option>
      </select>
    </div>
    <div class="task-filter-actions">
      <el-button type="primary" native-type="submit" :disabled="busy"
        >筛选</el-button
      ><el-button :disabled="busy" @click="emit('reset')">重置</el-button>
    </div>
  </form>
</template>
