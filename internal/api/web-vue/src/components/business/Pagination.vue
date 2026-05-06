<template>
  <div class="pagination">
    <button
      class="btn btn-ghost btn-sm"
      :disabled="currentPage <= 1"
      @click="$emit('change', -1)"
    >
      ← 上一页
    </button>

    <span class="page-info">
      第 {{ currentPage }} / {{ totalPages }} 页
      (共 {{ total }} 条)
    </span>

    <button
      class="btn btn-ghost btn-sm"
      :disabled="currentPage >= totalPages"
      @click="$emit('change', 1)"
    >
      下一页 →
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  currentPage: number
  totalPages: number
  total: number
}>()

defineEmits<{
  change: [delta: number]
}>()
</script>

<style scoped lang="scss">
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-4);
  padding: var(--space-4) 0;
}

.page-info {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  min-width: 150px;
  text-align: center;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  transition: all var(--duration-fast);

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);

  &:hover:not(:disabled) {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }
}

.btn-sm {
  padding: 6px 12px;
}
</style>
