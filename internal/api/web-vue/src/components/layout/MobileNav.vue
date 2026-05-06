<template>
  <nav class="mobile-nav">
    <div class="mobile-nav-items">
      <div
        v-for="item in navItems"
        :key="item.page"
        class="mobile-nav-item"
        :class="{ active: currentPage === item.page }"
        @click="$emit('navigate', item.page)"
      >
        <span class="icon">{{ item.icon }}</span>
        <span>{{ item.label }}</span>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
defineProps<{
  currentPage: string
}>()

defineEmits<{
  navigate: [page: string]
}>()

const navItems = [
  { page: 'overview', icon: '📊', label: '概览' },
  { page: 'tasks', icon: '🚀', label: '任务' },
  { page: 'data', icon: '📁', label: '数据' },
  { page: 'schedules', icon: '⏰', label: '定时' },
  { page: 'system', icon: '⚙️', label: '系统' }
]
</script>

<style scoped lang="scss">
.mobile-nav {
  display: none;
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: var(--mobile-nav-height);
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-primary);
  z-index: 100;
}

.mobile-nav-items {
  display: flex;
  justify-content: space-around;
  align-items: center;
  height: 100%;
}

.mobile-nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-1);
  padding: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--text-xs);
  min-width: 64px;
  cursor: pointer;

  &.active {
    color: var(--accent-primary);
  }

  .icon {
    font-size: var(--text-xl);
  }
}

@media (max-width: 767px) {
  .mobile-nav {
    display: block;
  }
}
</style>
