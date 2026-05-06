<template>
  <Teleport to="body">
    <div class="toast-container">
      <TransitionGroup name="toast" tag="div">
        <div
          v-for="item in toastList"
          :key="item.id"
          class="toast"
          :class="[`toast-${item.type}`]"
        >
          <div class="toast-icon">
            <span v-if="item.type === 'success'">✓</span>
            <span v-else-if="item.type === 'error'">✕</span>
            <span v-else-if="item.type === 'warning'">⚠</span>
            <span v-else>ℹ</span>
          </div>

          <div class="toast-content">
            <div class="toast-message">{{ item.message }}</div>
            <button
              v-if="item.action"
              class="toast-action"
              @click="handleAction(item)"
            >
              {{ item.action.label }}
            </button>
          </div>

          <button
            v-if="!item.duration || item.type === 'error'"
            class="toast-close"
            @click="handleRemove(item.id)"
          >
            ✕
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ToastMessage } from '@/composables/useToast'

const props = defineProps<{
  toasts: ToastMessage[]
}>()

const emit = defineEmits<{
  remove: [id: string]
}>()

const toastList = computed(() => props.toasts)

function handleRemove(id: string) {
  emit('remove', id)
}

function handleAction(item: ToastMessage) {
  if (item.action?.onClick) {
    item.action.onClick()
  }
}
</script>

<style scoped lang="scss">
.toast-container {
  position: fixed;
  bottom: var(--space-6);
  left: var(--space-6);
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  max-width: 420px;
  width: calc(100% - var(--space-12));
}

.toast {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  background: var(--bg-elevated);
  border: 1px solid var(--border-secondary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  backdrop-filter: blur(12px);
}

.toast-icon {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-sm);
  font-weight: var(--font-bold);
}

.toast-success .toast-icon {
  background: var(--success-bg);
  color: var(--success);
}

.toast-error .toast-icon {
  background: var(--danger-bg);
  color: var(--danger);
}

.toast-warning .toast-icon {
  background: var(--warning-bg);
  color: var(--warning);
}

.toast-info .toast-icon {
  background: var(--info-bg);
  color: var(--info);
}

.toast-content {
  flex: 1;
  min-width: 0;
}

.toast-message {
  font-size: var(--text-sm);
  line-height: 1.5;
  color: var(--text-primary);
}

.toast-action {
  margin-top: var(--space-2);
  padding: var(--space-1) var(--space-3);
  background: transparent;
  color: var(--accent-primary);
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
  border-radius: var(--radius-sm);

  &:hover {
    background: rgba(47, 129, 247, 0.1);
  }
}

.toast-close {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
  font-size: var(--text-xs);
  border-radius: var(--radius-sm);

  &:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }
}

// Transition animations
.toast-enter-active {
  animation: slideInRight 0.3s var(--ease-out);
}

.toast-leave-active {
  animation: slideOutRight 0.25s ease-in;
}

.toast-move {
  transition: transform 0.25s ease-in;
}

@keyframes slideInRight {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

@keyframes slideOutRight {
  from {
    transform: translateX(0);
    opacity: 1;
  }
  to {
    transform: translateX(100%);
    opacity: 0;
  }
}

@media (max-width: 767px) {
  .toast-container {
    left: var(--space-3);
    right: var(--space-3);
    bottom: calc(var(--mobile-nav-height) + var(--space-3));
    max-width: none;
  }

  .toast {
    padding: var(--space-3) var(--space-4);
  }
}
</style>
