<template>
  <Teleport to="body">
    <div class="drawer-overlay" :class="{ open: modelValue }" @click="close"></div>
    <aside class="drawer" :class="{ open: modelValue }">
      <div class="drawer-header">
        <h3 class="drawer-title">{{ title }}</h3>
        <button class="btn btn-ghost btn-icon" @click="close">✕</button>
      </div>
      <div class="drawer-body">
        <slot></slot>
      </div>
      <div v-if="$slots.footer || footer" class="drawer-footer">
        <slot name="footer">{{ footer }}</slot>
      </div>
    </aside>
  </Teleport>
</template>

<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
  title: string
  footer?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

function close() {
  emit('update:modelValue', false)
}
</script>

<style scoped lang="scss">
.drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  z-index: 200;
  opacity: 0;
  visibility: hidden;
  transition: all var(--duration-normal);

  &.open {
    opacity: 1;
    visibility: visible;
  }
}

.drawer {
  position: fixed;
  right: 0;
  top: 0;
  width: 420px;
  max-width: 100%;
  height: 100vh;
  background: var(--bg-secondary);
  border-left: 1px solid var(--border-primary);
  z-index: 201;
  transform: translateX(100%);
  transition: transform var(--duration-normal) var(--ease-out);
  display: flex;
  flex-direction: column;

  &.open {
    transform: translateX(0);
  }
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border-primary);
}

.drawer-title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-5);
}

.drawer-footer {
  padding: var(--space-4) var(--space-5);
  border-top: 1px solid var(--border-primary);
  display: flex;
  gap: var(--space-3);
  justify-content: flex-end;
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

@media (max-width: 767px) {
  .drawer {
    width: 100%;
  }
}
</style>
