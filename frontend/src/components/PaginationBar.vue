<script setup lang="ts">
defineProps<{
  page: number;
  total: number;
  pageSize?: number;
  busy?: boolean;
}>();
defineEmits<{ change: [page: number] }>();
</script>
<template>
  <div class="pagination-bar">
    <span>共 {{ total }} 条 · 第 {{ page }} 页</span>
    <div>
      <el-button
        :disabled="busy || page <= 1"
        @click="$emit('change', page - 1)"
        >上一页</el-button
      ><el-button
        :disabled="busy || page * (pageSize ?? 20) >= total"
        @click="$emit('change', page + 1)"
        >下一页</el-button
      >
    </div>
  </div>
</template>
