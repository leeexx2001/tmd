<template>
  <aside class="sidebar" :class="{ open: appStore.sidebarOpen }">
    <div class="sidebar-header">
      <div class="logo">
        <svg viewBox="0 0 100 100" aria-hidden="true">
          <defs>
            <linearGradient id="logoGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#2f81f7"/>
              <stop offset="100%" stop-color="#1f6feb"/>
            </linearGradient>
          </defs>
          <rect x="8" y="8" width="84" height="84" rx="20" ry="20" fill="url(#logoGrad)"/>
          <path d="M32 35 L68 35 L68 42 L52 42 L52 70 L45 70 L45 42 L32 42 Z" fill="white"/>
          <path d="M38 50 L62 50" stroke="white" stroke-width="4" stroke-linecap="round" opacity="0.6"/>
          <circle cx="72" cy="28" r="8" fill="#3fb950"/>
        </svg>
      </div>
      <span class="logo-text">TMD Pro</span>
    </div>

    <nav class="sidebar-nav">
      <div
        v-for="item in navItems"
        :key="item.page"
        class="nav-item"
        :class="{ active: currentPage === item.page }"
        @click="$emit('navigate', item.page)"
      >
        <span class="icon">{{ item.icon }}</span>
        <span>{{ item.label }}</span>
      </div>
    </nav>

    <div class="sidebar-footer">
      TMD Pro v2.0.0
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useAppStore } from '@/stores/appStore'

defineProps<{
  currentPage: string
}>()

defineEmits<{
  navigate: [page: string]
}>()

const appStore = useAppStore()

const navItems = [
  { page: 'overview', icon: '📊', label: '概览' },
  { page: 'tasks', icon: '🚀', label: '任务中心' },
  { page: 'data', icon: '📁', label: '数据管理' },
  { page: 'schedules', icon: '⏰', label: '定时任务' },
  { page: 'system', icon: '⚙️', label: '系统' }
]
</script>

<style scoped lang="scss">
.sidebar {
  position: fixed;
  left: 0;
  top: 0;
  width: var(--sidebar-width);
  height: 100vh;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-primary);
  display: flex;
  flex-direction: column;
  z-index: 100;
  transition: transform var(--duration-normal) var(--ease-out);
}

.sidebar-header {
  height: var(--header-height);
  display: flex;
  align-items: center;
  padding: 0 var(--space-4);
  border-bottom: 1px solid var(--border-primary);
  gap: var(--space-3);
}

.logo {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;

  svg {
    width: 100%;
    height: 100%;
    filter: drop-shadow(0 2px 4px rgba(47, 129, 247, 0.3));
  }
}

.logo-text {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
}

.sidebar-nav {
  flex: 1;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.nav-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: var(--text-base);
  font-weight: var(--font-medium);
  transition: all var(--duration-fast);
  cursor: pointer;

  &:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }

  &.active {
    background: var(--accent-primary);
    color: white;
  }
}

.nav-item .icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-lg);
}

.sidebar-footer {
  padding: var(--space-4);
  border-top: 1px solid var(--border-primary);
  font-size: var(--text-xs);
  color: var(--text-tertiary);
  text-align: center;
}

@media (max-width: 1023px) {
  .sidebar {
    transform: translateX(-100%);

    &.open {
      transform: translateX(0);
    }
  }
}
</style>
