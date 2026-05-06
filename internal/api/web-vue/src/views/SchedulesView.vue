<template>
  <div class="schedules-page">
    <!-- Warning Banner (if scheduler not running) -->
    <div v-if="!schedulerRunning" class="warning-banner">
      ⚠️ 调度器未启动 - 定时任务将不会自动执行
      <button class="btn btn-ghost btn-sm" @click="checkSchedulerStatus">
        检查状态
      </button>
    </div>

    <!-- Schedules List Card -->
    <div class="card">
      <div class="card-header">
        <div class="card-title">⏰ 定时任务列表</div>
        <button
          class="btn btn-primary btn-sm"
          @click="showCreateModal = true"
        >
          ➕ 创建定时任务
        </button>
      </div>

      <div class="card-body" style="padding: 0;">
        <!-- Loading State -->
        <div v-if="loading" class="loading-state">
          <div class="loading-spinner"></div>
          <span>加载定时任务...</span>
        </div>

        <!-- Empty State -->
        <div v-else-if="schedules.length === 0" class="empty-state">
          <div class="empty-icon">⏰</div>
          <div class="empty-title">暂无定时任务</div>
          <div class="empty-desc">创建一个定时规则来自动下载 Twitter 媒体文件</div>
        </div>

        <!-- Schedule Items List -->
        <div v-else class="schedule-list">
          <div
            v-for="(schedule, index) in schedules"
            :key="schedule.entry.id"
            class="schedule-item"
          >
            <div class="schedule-info">
              <div class="schedule-header">
                <h3 class="schedule-name">{{ schedule.entry.name || `Task ${index + 1}` }}</h3>
                <span :class="['status-badge', schedule.entry.enabled ? 'enabled' : 'disabled']">
                  {{ schedule.entry.enabled ? '✅ 启用' : '⛔ 禁用' }}
                </span>
              </div>

              <div class="schedule-details">
                <div class="detail-row">
                  <span class="detail-label">类型:</span>
                  <span class="detail-value">{{ getTypeLabel(schedule.entry.type) }}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">目标:</span>
                  <code class="detail-code">{{ schedule.entry.target }}</code>
                </div>
                <div class="detail-row">
                  <span class="detail-label">调度:</span>
                  <span class="detail-value">{{ schedule.schedule_display }}</span>
                </div>
              </div>

              <div class="schedule-stats">
                <div class="stat-item">
                  <span class="stat-number">{{ schedule.run_count }}</span>
                  <span class="stat-label">运行次数</span>
                </div>
                <div class="stat-item">
                  <span class="stat-number error" v-if="schedule.consecutive_failures > 0">
                    {{ schedule.consecutive_failures }}
                  </span>
                  <span class="stat-number" v-else>0</span>
                  <span class="stat-label">连续失败</span>
                </div>
                <div class="stat-item">
                  <span class="stat-label">下次执行:</span>
                  <span class="stat-time">{{ formatTime(schedule.next_run_at) }}</span>
                </div>
                <div class="stat-item">
                  <span class="stat-label">上次执行:</span>
                  <span class="stat-time">{{ formatTime(schedule.last_run_at) }}</span>
                </div>
              </div>
            </div>

            <div class="schedule-actions">
              <button
                class="btn btn-ghost btn-sm"
                title="手动触发"
                @click="handleTrigger(schedule.entry.id)"
                :disabled="triggeringId === schedule.entry.id"
              >
                {{ triggeringId === schedule.entry.id ? '⏳ 执行中...' : '▶️ 触发' }}
              </button>
              <button
                class="btn btn-ghost btn-sm"
                :title="schedule.entry.enabled ? '禁用' : '启用'"
                @click="handleToggle(schedule.entry.id, !schedule.entry.enabled)"
              >
                {{ schedule.entry.enabled ? '⛔ 禁用' : '✅ 启用' }}
              </button>
              <button
                class="btn btn-danger btn-sm"
                title="删除"
                @click="handleDelete(schedule.entry.id)"
              >
                🗑️ 删除
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal (Drawer) -->
    <Drawer
      v-model="showCreateModal"
      title="创建定时任务"
      footer="true"
    >
      <form @submit.prevent="handleCreateSchedule" class="schedule-form">
        <div class="form-group">
          <label class="form-label">任务名称（可选）</label>
          <input
            v-model="newSchedule.name"
            type="text"
            class="form-input"
            placeholder="例如: 每日备份 elonmusk"
          />
        </div>

        <div class="form-group">
          <label class="form-label">任务类型 *</label>
          <select v-model="newSchedule.type" class="form-select" required>
            <option value="">请选择类型</option>
            <option value="list">列表下载</option>
            <option value="user">用户下载</option>
            <option value="following">关注者下载</option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">目标 *</label>
          <input
            v-model="newSchedule.target"
            type="text"
            class="form-input"
            placeholder="用户名、List ID 或其他标识符"
            required
          />
        </div>

        <div class="form-group">
          <label class="form-label">Cron 表达式 *</label>
          <input
            v-model="newSchedule.schedule"
            type="text"
            class="form-input"
            placeholder='例如: "0 8 * * *" （每天早上8点）'
            required
          />
          <p class="field-hint">
            支持 Cron 格式，例如：
            <br/>• 0 8 * * * → 每天 08:00
            <br/>• */30 * * * * → 每30分钟
            <br/>• 0 9,18 * * * → 每天 09:00 和 18:00
          </p>
        </div>

        <div class="form-group checkbox-group">
          <label class="checkbox-label">
            <input v-model="newSchedule.run_on_start" type="checkbox" />
            启动时立即运行一次
          </label>
          <label class="checkbox-label">
            <input v-model="newSchedule.auto_follow" type="checkbox" />
            自动关注新用户
          </label>
          <label class="checkbox-label">
            <input v-model="newSchedule.skip_profile" type="checkbox" />
            跳过 Profile 下载
          </label>
          <label class="checkbox-label">
            <input v-model="newSchedule.no_retry" type="checkbox" />
            失败不重试
          </label>
        </div>

        <div class="form-actions">
          <button type="submit" class="btn btn-primary" :disabled="createLoading">
            {{ createLoading ? '创建中...' : '✨ 创建定时任务' }}
          </button>
          <button type="button" class="btn btn-secondary" @click="showCreateModal = false">
            取消
          </button>
        </div>
      </form>
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { schedulesApi } from '@/api/config'
import { toast } from '@/composables/useToast'
import Drawer from '@/components/layout/Drawer.vue'
import type { ScheduleStatus } from '@/types'

// State
const schedules = ref<ScheduleStatus[]>([])
const loading = ref(false)
const schedulerRunning = ref(true)
const showCreateModal = ref(false)
const createLoading = ref(false)
const triggeringId = ref<string | null>(null)

// Form state for new schedule
const newSchedule = reactive({
  name: '',
  type: '',
  target: '',
  schedule: '',
  run_on_start: false,
  auto_follow: true,
  skip_profile: false,
  no_retry: false
})

// Load data on mount
onMounted(async () => {
  await fetchSchedules()
})

async function fetchSchedules() {
  loading.value = true

  try {
    schedules.value = await schedulesApi.getSchedules()

    // Check if any schedules exist to determine if scheduler is running
    // This is a simplified check - in real implementation you'd call a health endpoint
    schedulerRunning.value = true
  } catch (error) {
    console.error('Failed to fetch schedules:', error)
    toast.error('获取定时任务失败')
  } finally {
    loading.value = false
  }
}

function checkSchedulerStatus() {
  // In a real implementation, this would call an API endpoint
  toast.info('检查调度器状态...')
}

function getTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    list: '📋 列表下载',
    user: '👤 用户下载',
    following: '🔗 关注者下载'
  }

  return labels[type] || type
}

function formatTime(timeStr: string | undefined): string {
  if (!timeStr) return '-'

  try {
    return new Date(timeStr).toLocaleString()
  } catch {
    return timeStr
  }
}

async function handleTrigger(id: string) {
  triggeringId.value = id

  try {
    await schedulesApi.triggerSchedule(id)
    toast.success('任务已触发执行')
  } catch (error: any) {
    toast.error(error.message || '触发失败')
  } finally {
    triggeringId.value = null
  }
}

async function handleToggle(id: string, enabled: boolean) {
  try {
    await schedulesApi.toggleSchedule(id, enabled)
    toast.success(enabled ? '任务已启用' : '任务已禁用')

    // Update local state optimistically
    const schedule = schedules.value.find(s => s.entry.id === id)
    if (schedule) {
      schedule.entry.enabled = enabled
    }
  } catch (error: any) {
    toast.error(error.message || '操作失败')
  }
}

async function handleDelete(id: string) {
  if (!confirm('确定要删除这个定时任务吗？')) return

  try {
    await schedulesApi.deleteSchedule(id)
    toast.success('定时任务已删除')

    // Remove from local list
    schedules.value = schedules.value.filter(s => s.entry.id !== id)
  } catch (error: any) {
    toast.error(error.message || '删除失败')
  }
}

async function handleCreateSchedule() {
  // Validate required fields
  if (!newSchedule.type || !newSchedule.target || !newSchedule.schedule) {
    toast.warning('请填写所有必填字段')
    return
  }

  createLoading.value = true

  try {
    await schedulesApi.createSchedule({
      ...newSchedule,
      enabled: true
    })

    toast.success('定时任务创建成功')

    // Reset form
    newSchedule.name = ''
    newSchedule.type = ''
    newSchedule.target = ''
    newSchedule.schedule = ''
    newSchedule.run_on_start = false
    newSchedule.auto_follow = true
    newSchedule.skip_profile = false
    newSchedule.no_retry = false

    showCreateModal.value = false

    // Refresh list
    await fetchSchedules()
  } catch (error: any) {
    toast.error(error.message || '创建失败')
  } finally {
    createLoading.value = false
  }
}
</script>

<style scoped lang="scss">
.schedules-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.warning-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  background: rgba(240, 136, 62, 0.12);
  border: 1px solid rgba(240, 136, 62, 0.4);
  border-radius: var(--radius-lg);
  color: var(--warning);
  font-size: var(--text-sm);
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

.card-body {
  padding: var(--space-5);
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-12) var(--space-6);
  text-align: center;
  gap: var(--space-3);
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
}

.empty-title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
}

.empty-desc {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  max-width: 300px;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--bg-tertiary);
  border-top-color: var(--accent-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

// Schedule items
.schedule-list {
  display: flex;
  flex-direction: column;
}

.schedule-item {
  display: flex;
  align-items: stretch;
  gap: var(--space-4);
  padding: var(--space-5);
  border-bottom: 1px solid var(--border-primary);

  &:last-child {
    border-bottom: none;
  }

  &:hover {
    background: var(--bg-tertiary);
  }
}

.schedule-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.schedule-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.schedule-name {
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin: 0;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: var(--radius-md);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);

  &.enabled {
    background: var(--success-bg);
    color: var(--success);
  }

  &.disabled {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
  }
}

.schedule-details {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-2) var(--space-6);
}

.detail-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
}

.detail-label {
  font-size: var(--text-xs);
  color: var(--text-tertiary);
  min-width: 40px;
}

.detail-value {
  font-size: var(--text-sm);
  color: var(--text-primary);
}

.detail-code {
  font-family: var(--font-mono);
  font-size: var(--text-xs);
  color: var(--info);
  background: var(--bg-primary);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.schedule-stats {
  display: flex;
  gap: var(--space-6);
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px dashed var(--border-secondary);
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-1);
}

.stat-number {
  font-size: var(--text-lg);
  font-weight: var(--font-bold);
  color: var(--text-primary);

  &.error {
    color: var(--danger);
  }
}

.stat-label,
.stat-time {
  font-size: var(--text-xs);
  color: var(--text-tertiary);
}

.stat-time {
  white-space: nowrap;
}

.schedule-actions {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  justify-content: center;
  padding-left: var(--space-4);
  border-left: 1px solid var(--border-primary);
}

// Create form styles
.schedule-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.form-label {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--text-secondary);
}

.form-input,
.form-select {
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
}

.form-select {
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%238b949e'%3E%3Cpath d='M6 9L1 4h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
}

.checkbox-group {
  flex-direction: row;
  flex-wrap: wrap;
  gap: var(--space-4);
}

.checkbox-label {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  font-size: var(--text-sm);
  color: var(--text-secondary);

  input[type="checkbox"] {
    width: 16px;
    height: 16px;
    accent-color: var(--accent-primary);
  }
}

.field-hint {
  font-size: var(--text-xs);
  color: var(--text-tertiary);
  line-height: 1.6;
}

.form-actions {
  display: flex;
  gap: var(--space-3);
  justify-content: flex-end;
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-primary);
}

// Common button styles
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

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

.btn-primary {
  background: var(--accent-primary);
  color: white;

  &:hover:not(:disabled) {
    background: var(--accent-hover);
  }
}

.btn-secondary {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: 1px solid var(--border-secondary);

  &:hover:not(:disabled) {
    background: var(--border-secondary);
  }
}

.btn-danger {
  background: transparent;
  color: var(--danger);
  border: 1px solid transparent;

  &:hover:not(:disabled) {
    background: var(--danger-bg);
  }
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);

  &:hover:not(:disabled) {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }
}

.btn-sm {
  padding: 6px 12px;
  font-size: var(--text-sm);
}

@media (max-width: 767px) {
  .schedule-item {
    flex-direction: column;
  }

  .schedule-actions {
    flex-direction: row;
    padding-left: 0;
    padding-top: var(--space-3);
    border-left: none;
    border-top: 1px solid var(--border-secondary);
  }

  .schedule-actions .btn {
    flex: 1;
  }

  .schedule-stats {
    flex-wrap: wrap;
    gap: var(--space-4);
  }

  .schedule-details {
    grid-template-columns: 1fr;
  }

  .form-actions {
    flex-direction: column-reverse;
  }

  .form-actions .btn {
    width: 100%;
  }
}
</style>
