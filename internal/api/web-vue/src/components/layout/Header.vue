<template>
  <header class="top-header">
    <div class="header-left">
      <button class="menu-toggle" @click="$emit('toggle-sidebar')">
        ☰
      </button>
      <h1 class="page-title">{{ title }}</h1>
    </div>

    <div class="header-actions">
      <span
        class="sse-indicator"
        :class="{ connected: sseConnected }"
        :title="sseConnected ? '实时连接正常' : '实时连接已断开'"
      >
        <span class="sse-dot"></span>
      </span>
      <button class="btn btn-ghost btn-icon" @click="$emit('refresh')" title="刷新">
        🔄
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  sseConnected: boolean
}>()

defineEmits<{
  refresh: []
  'toggle-sidebar': []
}>()
</script>

<style scoped lang="scss">
.top-header {
  height: var(--header-height);
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-primary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--space-6);
  position: sticky;
  top: 0;
  z-index: 50;
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.menu-toggle {
  display: none;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  font-size: var(--text-xl);

  &:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }
}

.page-title {
  font-size: var(--text-xl);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.sse-indicator {
  display: inline-flex;
  align-items: center;
  padding: 4px 8px;
  border-radius: var(--radius-md);
  cursor: default;
  transition: background 0.2s;

  &:hover {
    background: var(--bg-tertiary);
  }
}

.sse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--danger);
  transition: background 0.3s;
}

.sse-indicator.connected .sse-dot {
  background: var(--success);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: 10px 16px;
  border-radius: var(--radius-md);
  font-size: var(--text-base);
  font-weight: var(--font-medium);
  transition: all var(--duration-fast);
  cursor: pointer;
  white-space: nowrap;
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);

  &:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }
}

.btn-icon {
  width: 36px;
  height: 36px;
  padding: 0;
  border-radius: var(--radius-md);
}

@media (max-width: 1023px) {
  .menu-toggle {
    display: flex;
  }
}
</style>
