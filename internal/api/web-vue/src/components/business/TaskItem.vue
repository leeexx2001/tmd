<template>
  <div class="task-item" @click="showDetail">
    <div class="task-info">
      <div class="task-title">{{ task.type }} - {{ target }}</div>
      <div class="task-meta">
        <span :class="['tag', `tag-${task.status}`]">{{ statusText }}</span>
        <span>ID: {{ task.task_id }}</span>
        <span>{{ formatDate(task.created_at) }}</span>
      </div>
    </div>
    <div class="task-progress">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: `${progressPercent}%` }"></div>
      </div>
      <div class="task-progress-text">{{ progressPercent }}%{{ stageText }}</div>
    </div>
    <div class="task-actions" @click.stop>
      <button
        v-if="isRunning"
        class="btn btn-danger btn-sm"
        @click="handleCancel"
      >
        取消
      </button>
      <button
        v-else
        class="btn btn-ghost btn-sm"
        @click="showDetail"
      >
        详情
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Task } from '@/types'
import { useTaskStore } from '@/stores/taskStore'

const props = defineProps<{
  task: Task
}>()

const taskStore = useTaskStore()

const statusMap: Record<string, { tag: string; text: string }> = {
  queued: { tag: 'tag-queued', text: '排队' },
  running: { tag: 'tag-running', text: '运行' },
  completed: { tag: 'tag-completed', text: '完成' },
  failed: { tag: 'tag-failed', text: '失败' },
  cancelled: { tag: 'tag-cancelled', text: '取消' }
}

const statusText = computed(() => statusMap[props.task.status]?.text || props.task.status)
const isRunning = computed(() => props.task.status === 'running' || props.task.status === 'queued')

const target = computed(() => {
  const data = props.task.data || {}
  if (data.screen_name) return `@${data.screen_name}`
  if (data.list_id) return `List ${data.list_id}`

  const parts: string[] = []
  if (Array.isArray(data.users) && data.users.length) parts.push(`${data.users.length} 用户`)
  if (Array.isArray(data.lists) && data.lists.length) parts.push(`${data.lists.length} 列表`)
  return parts.length ? parts.join(' · ') : 'Unknown'
})

const progressPercent = computed(() => {
  if (props.task.status === 'completed') return 100

  const progress = props.task.progress || {}
  const total = progress.total || 0
  const completed = progress.completed || 0
  const ratio = total > 0 ? Math.min(completed / total, 1) : 0

  if (props.task.status === 'failed' || props.task.status === 'cancelled') {
    return total > 0 ? Math.round(ratio * 100) : 0
  }

  const stageMap: Record<string, number> = {
    syncing: 5,
    preparing: 10,
    downloading: Math.round(10 + ratio * 70),
    retrying: Math.round(80 + ratio * 10),
    profile: total > 0 ? Math.round(90 + ratio * 9) : 90,
    marking: total > 0 ? Math.round(10 + ratio * 85) : 10,
    completed: 100
  }

  return stageMap[progress.stage || ''] || 0
})

const stageText = computed(() => {
  const stage = props.task.progress?.stage
  if (!stage) return ''

  const stageMap: Record<string, string> = {
    preparing: ' · 准备中',
    syncing: ' · 同步列表',
    downloading: ' · 下载中',
    retrying: ' · 重试中',
    profile: ' · 下载资料',
    marking: ' · 标记中',
    completed: ''
  }

  return stageMap[stage] || (stage ? ` · ${stage}` : '')
})

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString()
}

function showDetail() {
  // Will implement drawer detail view later
  console.log('Show task detail:', props.task.task_id)
}

async function handleCancel() {
  if (!confirm('确定要取消这个任务吗？')) return

  try {
    await taskStore.cancelTask(props.task.task_id)
    alert('任务已取消')
  } catch (error) {
    console.error('Failed to cancel task:', error)
    alert('取消失败')
  }
}
</script>

<style scoped lang="scss">
.task-item {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4);
  border-bottom: 1px solid var(--border-primary);
  transition: background var(--duration-fast);
  cursor: pointer;

  &:hover {
    background: var(--bg-tertiary);
  }
}

.task-info {
  flex: 1;
  min-width: 0;
}

.task-title {
  font-weight: var(--font-medium);
  color: var(--text-primary);
  margin-bottom: var(--space-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.task-meta {
  font-size: var(--text-xs);
  color: var(--text-tertiary);
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.tag {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: 3px 10px;
  border-radius: var(--radius-md);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.tag-queued {
  background: var(--bg-tertiary);
  color: var(--text-secondary);
}

.tag-running {
  background: var(--info-bg);
  color: var(--info);

  &::before {
    content: '';
    width: 6px;
    height: 6px;
    background: currentColor;
    border-radius: 50%;
    animation: pulse 2s infinite;
    margin-right: 4px;
  }
}

.tag-completed {
  background: var(--success-bg);
  color: var(--success);
}

.tag-failed {
  background: var(--danger-bg);
  color: var(--danger);
}

.tag-cancelled {
  background: var(--warning-bg);
  color: var(--warning);
}

.task-progress {
  width: 120px;
  flex-shrink: 0;
}

.progress-bar {
  width: 100%;
  height: 6px;
  background: var(--bg-tertiary);
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-primary), var(--success));
  border-radius: 3px;
  transition: width 0.3s var(--ease-out);
}

.task-progress-text {
  font-size: var(--text-xs);
  color: var(--text-secondary);
  text-align: right;
  margin-top: var(--space-1);
}

.task-status,
.task-actions {
  flex-shrink: 0;
}

.task-actions {
  display: flex;
  gap: var(--space-2);
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

.btn-danger {
  background: transparent;
  color: var(--danger);
  border: 1px solid var(--danger);

  &:hover {
    background: var(--danger-bg);
  }
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);

  &:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }
}

.btn-sm {
  padding: 6px 12px;
  font-size: var(--text-sm);
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
