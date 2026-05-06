<template>
  <div class="overview-page">
    <!-- Stats Grid -->
    <div class="stats-grid">
      <StatCard
        icon="●"
        :color="'var(--success)'"
        :value="healthStatus"
        label="系统状态"
      />
      <StatCard
        icon="🚀"
        :color="'var(--info)'"
        :value="taskStore.taskStats.running.toString()"
        label="运行中任务"
      />
      <StatCard
        icon="✓"
        :color="'var(--success)'"
        :value="taskStore.taskStats.completed.toString()"
        label="已完成任务"
      />
    </div>

    <!-- Quick Download -->
    <div class="card">
      <div class="card-header">
        <div>
          <div class="card-title">⚡ 快速下载</div>
          <div class="card-subtitle">输入 Twitter 用户名或链接快速创建下载任务</div>
        </div>
      </div>
      <div class="card-body">
        <div style="display: flex; gap: var(--space-3); flex-wrap: wrap;">
          <input
            v-model="quickDownloadInput"
            type="text"
            class="form-input"
            placeholder="输入用户名，如: elonmusk 或 https://twitter.com/elonmusk"
            style="flex: 1; min-width: 280px;"
            @keypress.enter="handleQuickDownload"
          />
          <button class="btn btn-primary" @click="handleQuickDownload">创建任务</button>
        </div>
        <div class="text-sm text-tertiary" style="margin-top: var(--space-4);">
          支持格式: twitter.com/username | x.com/username | @username
        </div>
      </div>
    </div>

    <!-- Recent Tasks -->
    <div class="card">
      <div class="card-header">
        <div class="card-title">最近任务</div>
        <button class="btn btn-ghost btn-sm" @click="$router.push('/tasks')">查看全部 →</button>
      </div>
      <div class="card-body" style="padding: 0;">
        <div v-if="taskStore.recentTasks.length === 0" class="empty-state">
          <div class="empty-icon">📋</div>
          <div class="empty-title">暂无任务</div>
          <div class="empty-desc">创建一个新任务开始下载 Twitter 媒体文件</div>
        </div>
        <div v-else class="task-list">
          <TaskItem
            v-for="task in taskStore.recentTasks"
            :key="task.task_id"
            :task="task"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { toast } from '@/composables/useToast'
import StatCard from '@/components/business/StatCard.vue'
import TaskItem from '@/components/business/TaskItem.vue'

const taskStore = useTaskStore()

const quickDownloadInput = ref('')
const health = ref<any>(null)

const healthStatus = computed(() => {
  if (!health.value) return '检查中'
  return health.value.status === 'ok' ? '健康' : '异常'
})

async function handleQuickDownload() {
  const value = quickDownloadInput.value.trim()
  if (!value) return

  let username = value
  const match = value.match(/(?:twitter\.com|x\.com)\/([^/\s?]+)/)
  if (match) username = match[1]
  if (username.startsWith('@')) username = username.slice(1)

  try {
    await import('@/api/tasks').then(({ tasksApi }) =>
      tasksApi.createUserDownload(username, { auto_follow: true })
    )
    quickDownloadInput.value = ''
    toast.success(`已创建用户下载任务: @${username}`)
  } catch (error) {
    console.error('Failed to create task:', error)
    toast.error('创建任务失败')
  }
}

onMounted(async () => {
  try {
    const [healthData] = await Promise.all([
      import('@/api/tasks').then(({ tasksApi }) => tasksApi.getHealth()),
      taskStore.fetchTasks()
    ])
    health.value = healthData
  } catch (error) {
    console.error('Failed to load initial data:', error)
  }
})
</script>

<style scoped lang="scss">
.overview-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-4);
}

.card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border-primary);
  min-height: 56px;
}

.card-title {
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
}

.card-subtitle {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  margin-top: var(--space-1);
}

.card-body {
  padding: var(--space-5);
}

.form-input {
  width: 100%;
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  padding: 10px 12px;
  color: var(--text-primary);
  font-size: var(--text-base);
  transition: all var(--duration-fast);

  &:focus {
    border-color: var(--accent-primary);
    box-shadow: 0 0 0 3px rgba(47, 129, 247, 0.1);
  }

  &::placeholder {
    color: var(--text-tertiary);
  }
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

.btn-primary {
  background: var(--accent-primary);
  color: white;

  &:hover {
    background: var(--accent-hover);
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

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-12) var(--space-6);
  text-align: center;
}

.empty-icon {
  width: 64px;
  height: 64px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-xl);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-2xl);
  margin-bottom: var(--space-4);
}

.empty-title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin-bottom: var(--space-2);
}

.empty-desc {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  max-width: 300px;
}

.task-list {
  display: flex;
  flex-direction: column;
}

.text-sm {
  font-size: var(--text-sm);
}

.text-tertiary {
  color: var(--text-tertiary);
}
</style>
