<template>
  <div class="empty-state" :class="{ centered }">
    <div v-if="icon" class="empty-icon">{{ icon }}</div>
    <h3 v-if="title" class="empty-title">{{ title }}</h3>
    <p v-if="description" class="empty-description">{{ description }}</p>
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  icon?: string
  title?: string
  description?: string
  centered?: boolean
}>()
</script>

<style scoped lang="scss">
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-12) var(--space-6);
  text-align: center;
  
  &.centered {
    min-height: 300px;
  }
}

.empty-icon {
  font-size: 48px;
  margin-bottom: var(--space-4);
  opacity: 0.6;
}

.empty-title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin-bottom: var(--space-2);
}

.empty-description {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  max-width: 400px;
  line-height: 1.6;
}
</style>
