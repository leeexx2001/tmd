<template>
  <div class="error-boundary" v-if="error">
    <div class="error-content">
      <div class="error-icon">⚠️</div>
      <h3 class="error-title">出现了一些问题</h3>
      <p class="error-message">{{ error.message || '未知错误' }}</p>
      <div class="error-actions">
        <button class="btn btn-primary" @click="$emit('retry')">
          🔄 重试
        </button>
        <button class="btn btn-secondary" @click="dismiss">
          关闭
        </button>
      </div>
      <details v-if="showDetails" class="error-details">
        <summary>技术详情</summary>
        <pre>{{ error.stack || JSON.stringify(error, null, 2) }}</pre>
      </details>
    </div>
  </div>
  <slot v-else />
</template>

<script setup lang="ts">
defineProps<{
  error: Error | null
  showDetails?: boolean
}>()

const emit = defineEmits<{
  retry: []
  dismiss: []
}>()

function dismiss() {
  emit('dismiss')
}
</script>

<style scoped lang="scss">
.error-boundary {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  padding: var(--space-6);
}

.error-content {
  max-width: 500px;
  text-align: center;
}

.error-icon {
  font-size: 48px;
  margin-bottom: var(--space-4);
}

.error-title {
  font-size: var(--text-xl);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin-bottom: var(--space-3);
}

.error-message {
  font-size: var(--text-base);
  color: var(--danger);
  margin-bottom: var(--space-6);
  line-height: 1.6;
}

.error-actions {
  display: flex;
  gap: var(--space-3);
  justify-content: center;
  margin-bottom: var(--space-4);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: 10px 20px;
  border-radius: var(--radius-md);
  font-size: var(--text-base);
  font-weight: var(--font-medium);
  transition: all var(--duration-fast);
  cursor: pointer;
  border: none;
}

.btn-primary {
  background: var(--accent-primary);
  color: white;

  &:hover {
    background: var(--accent-hover);
  }
}

.btn-secondary {
  background: var(--bg-tertiary);
  color: var(--text-secondary);

  &:hover {
    background: var(--border-primary);
    color: var(--text-primary);
  }
}

.error-details {
  margin-top: var(--space-4);
  text-align: left;

  summary {
    cursor: pointer;
    color: var(--text-tertiary);
    font-size: var(--text-sm);
    margin-bottom: var(--space-2);
    
    &:hover {
      color: var(--text-secondary);
    }
  }

  pre {
    background: var(--bg-tertiary);
    border: 1px solid var(--border-primary);
    border-radius: var(--radius-md);
    padding: var(--space-4);
    overflow-x: auto;
    font-size: var(--text-xs);
    line-height: 1.5;
    color: var(--text-secondary);
    max-height: 200px;
    overflow-y: auto;
  }
}
</style>
