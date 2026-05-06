<template>
  <div class="loading-spinner" :class="[size, { centered }]">
    <div class="spinner"></div>
    <span v-if="text" class="loading-text">{{ text }}</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  size?: 'sm' | 'md' | 'lg'
  text?: string
  centered?: boolean
}>()
</script>

<style scoped lang="scss">
.loading-spinner {
  display: inline-flex;
  align-items: center;
  gap: var(--space-3);
  
  &.centered {
    justify-content: center;
    padding: var(--space-8);
    width: 100%;
    flex-direction: column;
  }
}

.spinner {
  width: 24px;
  height: 24px;
  border: 3px solid var(--border-primary);
  border-top-color: var(--accent-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.loading-text {
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.size-sm .spinner {
  width: 16px;
  height: 16px;
  border-width: 2px;
}

.size-lg .spinner {
  width: 32px;
  height: 32px;
  border-width: 4px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
