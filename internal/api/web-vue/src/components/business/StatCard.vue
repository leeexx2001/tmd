<template>
  <div class="stat-card">
    <div class="stat-icon" :style="{ color }">
      {{ icon }}
    </div>
    <div class="stat-content">
      <div class="stat-value">{{ value }}</div>
      <div class="stat-label">{{ label }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  icon: string
  color: string
  value: string
  label: string
}>()
</script>

<style scoped lang="scss">
.stat-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.stat-icon {
  width: 48px;
  height: 48px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-xl);
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: var(--text-2xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
  line-height: 1.2;
}

.stat-label {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  margin-top: var(--space-1);
}
</style>
