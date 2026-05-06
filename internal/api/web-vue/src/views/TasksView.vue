<template>
  <div class="tasks-page">
    <!-- Task Type Tabs -->
    <div class="task-tabs">
      <button
        v-for="tab in taskTabs"
        :key="tab.id"
        class="tab-btn"
        :class="{ active: activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        {{ tab.icon }} {{ tab.label }}
      </button>
    </div>

    <!-- Task Creation Form -->
    <div class="card form-card">
      <div class="card-header">
        <div class="card-title">📝 {{ currentTabLabel }}</div>
      </div>
      <div class="card-body">
        <!-- User Download Form -->
        <form v-if="activeTab === 'user'" @submit.prevent="handleCreateUserTask" class="task-form">
          <div class="form-group">
            <label class="form-label">Screen Name</label>
            <input
              v-model="forms.user.screenName"
              type="text"
              class="form-input"
              placeholder="例如: elonmusk"
              required
            />
          </div>
          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input v-model="forms.user.autoFollow" type="checkbox" />
              AutoFollow
            </label>
            <label class="checkbox-label">
              <input v-model="forms.user.skipProfile" type="checkbox" />
              SkipProfile
            </label>
            <label class="checkbox-label">
              <input v-model="forms.user.noRetry" type="checkbox" />
              NoRetry
            </label>
          </div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary">
              创建下载任务
            </button>
            <button type="button" class="btn btn-secondary" @click="handleCreateUserProfileTask">
              仅下载 Profile
            </button>
          </div>
        </form>

        <!-- List Download Form -->
        <form v-else-if="activeTab === 'list'" @submit.prevent="handleCreateListTask" class="task-form">
          <div class="form-group">
            <label class="form-label">List ID</label>
            <input
              v-model="forms.list.listId"
              type="number"
              class="form-input"
              placeholder="例如: 123456789"
              required
            />
          </div>
          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input v-model="forms.list.autoFollow" type="checkbox" />
              AutoFollow
            </label>
            <label class="checkbox-label">
              <input v-model="forms.list.skipProfile" type="checkbox" />
              SkipProfile
            </label>
            <label class="checkbox-label">
              <input v-model="forms.list.noRetry" type="checkbox" />
              NoRetry
            </label>
          </div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary">
              创建下载任务
            </button>
            <button type="button" class="btn btn-secondary" @click="handleCreateListProfileTask">
              仅下载 Profile
            </button>
          </div>
        </form>

        <!-- Following Download Form -->
        <form v-else-if="activeTab === 'following'" @submit.prevent="handleCreateFollowingTask" class="task-form">
          <div class="form-group">
            <label class="form-label">Screen Name</label>
            <input
              v-model="forms.following.screenName"
              type="text"
              class="form-input"
              placeholder="例如: elonmusk"
              required
            />
          </div>
          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input v-model="forms.following.autoFollow" type="checkbox" />
              AutoFollow
            </label>
            <label class="checkbox-label">
              <input v-model="forms.following.skipProfile" type="checkbox" />
              SkipProfile
            </label>
            <label class="checkbox-label">
              <input v-model="forms.following.noRetry" type="checkbox" />
              NoRetry
            </label>
          </div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary">
              创建关注下载任务
            </button>
          </div>
        </form>

        <!-- Mark Form -->
        <form v-else-if="activeTab === 'mark'" @submit.prevent="handleCreateMarkTask" class="task-form">
          <div class="form-group">
            <label class="form-label">用户 Screen Name（每行一个）</label>
            <textarea
              v-model="forms.mark.users"
              class="form-textarea"
              placeholder="elonmusk&#10;jack"
              rows="3"
            ></textarea>
          </div>
          <div class="form-group">
            <label class="form-label">List IDs（每行一个）</label>
            <textarea
              v-model="forms.mark.lists"
              class="form-textarea"
              placeholder="123456789&#10;987654321"
              rows="3"
            ></textarea>
          </div>
          <div class="form-group">
            <label class="form-label">Following 用户（每行一个）</label>
            <textarea
              v-model="forms.mark.followingNames"
              class="form-textarea"
              placeholder="user_a&#10;user_b"
              rows="3"
            ></textarea>
          </div>
          <div class="form-group">
            <label class="form-label">标记时间（可选）</label>
            <input
              v-model="forms.mark.timestamp"
              type="datetime-local"
              class="form-input"
            />
            <div class="form-hint">留空则使用服务器当前时间。每个输入目标会创建独立标记任务。</div>
          </div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary">
              创建标记任务
            </button>
          </div>
        </form>

        <!-- Batch Download Form -->
        <form v-else-if="activeTab === 'batch'" @submit.prevent="handleCreateBatchTask" class="task-form">
          <div class="form-group">
            <label class="form-label">用户列表（每行一个）</label>
            <textarea
              v-model="forms.batch.users"
              class="form-textarea"
              placeholder="user1&#10;user2&#10;user3"
              rows="4"
            ></textarea>
          </div>
          <div class="form-group">
            <label class="form-label">List IDs（每行一个）</label>
            <textarea
              v-model="forms.batch.lists"
              class="form-textarea"
              placeholder="123&#10;456&#10;789"
              rows="3"
            ></textarea>
          </div>
          <div class="form-group">
            <label class="form-label">Following 用户（每行一个）</label>
            <textarea
              v-model="forms.batch.followingNames"
              class="form-textarea"
              placeholder="user_a&#10;user_b"
              rows="3"
            ></textarea>
            <div class="form-hint">将这些用户的 Following 加入批量下载目标</div>
          </div>
          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input v-model="forms.batch.autoFollow" type="checkbox" />
              AutoFollow
            </label>
            <label class="checkbox-label">
              <input v-model="forms.batch.skipProfile" type="checkbox" />
              SkipProfile
            </label>
            <label class="checkbox-label">
              <input v-model="forms.batch.noRetry" type="checkbox" />
              NoRetry
            </label>
          </div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary">
              创建批量任务
            </button>
          </div>
        </form>

        <!-- JSON File Form -->
        <form v-else-if="activeTab === 'jsonfile'" @submit.prevent="handleCreateJsonFileTask" class="task-form">
          <div class="form-group">
            <label class="form-label">第三方工具导出的JSON文件路径（每行一个）</label>
            <textarea
              v-model="forms.jsonfile.paths"
              class="form-textarea"
              placeholder="/path/to/twitter-followers-123.json&#10;/path/to/more.json"
              rows="4"
            ></textarea>
          </div>
          <div class="form-hint">支持格式: 第三方工具导出的Twitter推文搜索结果JSON（含推文列表、media数组、metadata字段）</div>
          <div class="form-group checkbox-group mt-3">
            <label class="checkbox-label">
              <input v-model="forms.jsonfile.noRetry" type="checkbox" />
              NoRetry
            </label>
          </div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary">
              创建 JSON 文件任务
            </button>
          </div>
        </form>

        <!-- JSON Folder Form -->
        <form v-else-if="activeTab === 'jsonfolder'" @submit.prevent="handleCreateJsonFolderTask" class="task-form">
          <div class="form-group">
            <label class="form-label">TMD .loongtweet 文件夹路径（每行一个）</label>
            <textarea
              v-model="forms.jsonfolder.paths"
              class="form-textarea"
              placeholder="/path/to/.loongtweet&#10;/path/to/another/.loongtweet"
              rows="4"
            ></textarea>
          </div>
          <div class="form-hint">从 TMD 生成的 .loongtweet 目录下载推文媒体文件（仅下载媒体，不保存元数据）</div>
          <div class="form-group checkbox-group mt-3">
            <label class="checkbox-label">
              <input v-model="forms.jsonfolder.noRetry" type="checkbox" />
              NoRetry
            </label>
          </div>
          <div class="form-actions">
            <button type="submit" class="btn btn-primary">
              创建 LoongTweet 任务
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Task List Section -->
    <div class="card tasks-card">
      <div class="card-header">
        <div class="card-title">📋 任务列表 ({{ taskStore.filteredTasks.length }})</div>
        <div class="tasks-toolbar">
          <select
            v-model="taskStore.taskFilter"
            class="form-select"
          >
            <option value="all">全部状态</option>
            <option value="running">运行中</option>
            <option value="queued">排队中</option>
            <option value="completed">已完成</option>
            <option value="failed">失败</option>
          </select>
          <input
            v-model="taskStore.taskSearch"
            type="text"
            class="form-input search-input"
            placeholder="搜索任务..."
          />
        </div>
      </div>
      <div class="card-body" style="padding: 0;">
        <div v-if="taskStore.loading" class="loading-state">
          <div class="loading-spinner"></div>
          <span>加载中...</span>
        </div>
        <div v-else-if="taskStore.filteredTasks.length === 0" class="empty-state">
          <div class="empty-icon">📋</div>
          <div class="empty-title">{{ taskStore.tasks.length === 0 ? '暂无任务' : '没有匹配的任务' }}</div>
          <div class="empty-desc">
            {{ taskStore.tasks.length === 0 ? '创建一个新任务开始下载 Twitter 媒体文件' : '尝试调整筛选条件或搜索关键词' }}
          </div>
        </div>
        <div v-else class="task-list">
          <TaskItem
            v-for="task in taskStore.filteredTasks"
            :key="task.task_id"
            :task="task"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { toast } from '@/composables/useToast'
import { tasksApi } from '@/api/tasks'
import TaskItem from '@/components/business/TaskItem.vue'

const taskStore = useTaskStore()

// Task tabs configuration
const taskTabs = [
  { id: 'user', label: '用户下载', icon: '👤' },
  { id: 'list', label: '列表下载', icon: '📋' },
  { id: 'following', label: '关注下载', icon: '🔗' },
  { id: 'mark', label: '标记', icon: '🏷️' },
  { id: 'batch', label: '批量下载', icon: '📦' },
  { id: 'jsonfile', label: 'JSON文件', icon: '📄' },
  { id: 'jsonfolder', label: 'LoongTweet', icon: '📁' }
]

const activeTab = ref('user')

const currentTabLabel = computed(() => {
  return taskTabs.find(t => t.id === activeTab.value)?.label || ''
})

// Form states for each task type
const forms = reactive({
  user: {
    screenName: '',
    autoFollow: true,
    skipProfile: false,
    noRetry: false
  },
  list: {
    listId: '',
    autoFollow: true,
    skipProfile: false,
    noRetry: false
  },
  following: {
    screenName: '',
    autoFollow: true,
    skipProfile: false,
    noRetry: false
  },
  mark: {
    users: '',
    lists: '',
    followingNames: '',
    timestamp: ''
  },
  batch: {
    users: '',
    lists: '',
    followingNames: '',
    autoFollow: true,
    skipProfile: false,
    noRetry: false
  },
  jsonfile: {
    paths: '',
    noRetry: false
  },
  jsonfolder: {
    paths: '',
    noRetry: false
  }
})

// Task creation handlers
async function handleCreateUserTask() {
  const screenName = forms.user.screenName.trim()
  if (!screenName) {
    toast.warning('请输入用户名')
    return
  }

  try {
    await tasksApi.createUserDownload(screenName, {
      auto_follow: forms.user.autoFollow,
      skip_profile: forms.user.skipProfile,
      no_retry: forms.user.noRetry
    })
    toast.success(`已创建用户下载任务: @${screenName}`)
    resetForm('user')
  } catch (error: any) {
    toast.error(error.message || '创建任务失败')
  }
}

async function handleCreateUserProfileTask() {
  const screenName = forms.user.screenName.trim()
  if (!screenName) {
    toast.warning('请输入用户名')
    return
  }

  try {
    await tasksApi.createProfileDownload(screenName)
    toast.success(`已创建 Profile 下载任务: @${screenName}`)
    resetForm('user')
  } catch (error: any) {
    toast.error(error.message || '创建任务失败')
  }
}

async function handleCreateListTask() {
  const listId = forms.list.listId.trim()
  if (!listId) {
    toast.warning('请输入 List ID')
    return
  }

  try {
    await tasksApi.createListDownload(listId, {
      auto_follow: forms.list.autoFollow,
      skip_profile: forms.list.skipProfile,
      no_retry: forms.list.noRetry
    })
    toast.success(`已创建列表下载任务: List ${listId}`)
    resetForm('list')
  } catch (error: any) {
    toast.error(error.message || '创建任务失败')
  }
}

async function handleCreateListProfileTask() {
  const listId = forms.list.listId.trim()
  if (!listId) {
    toast.warning('请输入 List ID')
    return
  }

  try {
    await tasksApi.createListProfile(listId)
    toast.success(`已创建列表 Profile 下载: List ${listId}`)
    resetForm('list')
  } catch (error: any) {
    toast.error(error.message || '创建任务失败')
  }
}

async function handleCreateFollowingTask() {
  const screenName = forms.following.screenName.trim()
  if (!screenName) {
    toast.warning('请输入用户名')
    return
  }

  try {
    await tasksApi.createFollowingDownload(screenName, {
      auto_follow: forms.following.autoFollow,
      skip_profile: forms.following.skipProfile,
      no_retry: forms.following.noRetry
    })
    toast.success(`已创建关注下载任务: @${screenName} 的关注者`)
    resetForm('following')
  } catch (error: any) {
    toast.error(error.message || '创建任务失败')
  }
}

async function handleCreateMarkTask() {
  const users = forms.mark.users.split('\n').filter(u => u.trim())
  const lists = forms.mark.lists.split('\n').map(l => parseInt(l.trim())).filter(l => !isNaN(l))
  const followingNames = forms.mark.followingNames.split('\n').filter(f => f.trim())

  if (users.length === 0 && lists.length === 0 && followingNames.length === 0) {
    toast.warning('请至少填写一种标记目标')
    return
  }

  let createdCount = 0

  try {
    // Create mark tasks for each target
    for (const user of users) {
      await tasksApi.createUserMark(user.trim(), forms.mark.timestamp)
      createdCount++
    }
    for (const list of lists) {
      await tasksApi.createListMark(list, forms.mark.timestamp)
      createdCount++
    }
    for (const name of followingNames) {
      await tasksApi.createFollowingMark(name.trim(), forms.mark.timestamp)
      createdCount++
    }

    toast.success(`已创建 ${createdCount} 个标记任务`)
    resetForm('mark')
  } catch (error: any) {
    toast.error(error.message || '创建标记任务失败')
  }
}

async function handleCreateBatchTask() {
  const users = forms.batch.users.split('\n').filter(u => u.trim())
  const lists = forms.batch.lists.split('\n').map(l => parseInt(l.trim())).filter(l => !isNaN(l))
  const followingNames = forms.batch.followingNames.split('\n').filter(f => f.trim())

  if (users.length === 0 && lists.length === 0 && followingNames.length === 0) {
    toast.warning('请至少填写一种批量下载目标')
    return
  }

  try {
    await tasksApi.createBatchDownload({
      users,
      lists,
      following_names: followingNames,
      auto_follow: forms.batch.autoFollow,
      skip_profile: forms.batch.skipProfile,
      no_retry: forms.batch.noRetry
    })

    toast.success('已创建批量下载任务')
    resetForm('batch')
  } catch (error: any) {
    toast.error(error.message || '创建批量任务失败')
  }
}

async function handleCreateJsonFileTask() {
  const paths = forms.jsonfile.paths.split('\n').filter(p => p.trim())

  if (paths.length === 0) {
    toast.warning('请至少填写一个 JSON 文件路径')
    return
  }

  try {
    await tasksApi.createJsonFileDownload({
      paths,
      no_retry: forms.jsonfile.noRetry
    })

    toast.success(`已创建 ${paths.length} 个 JSON 文件下载任务`)
    resetForm('jsonfile')
  } catch (error: any) {
    toast.error(error.message || '创建 JSON 文件任务失败')
  }
}

async function handleCreateJsonFolderTask() {
  const paths = forms.jsonfolder.paths.split('\n').filter(p => p.trim())

  if (paths.length === 0) {
    toast.warning('请至少填写一个文件夹路径')
    return
  }

  try {
    await tasksApi.createJsonFolderDownload({
      paths,
      no_retry: forms.jsonfolder.noRetry
    })

    toast.success(`已创建 ${paths.length} 个 LoongTweet 下载任务`)
    resetForm('jsonfolder')
  } catch (error: any) {
    toast.error(error.message || '创建 LoongTweet 任务失败')
  }
}

function resetForm(type: string) {
  switch (type) {
    case 'user':
      forms.user.screenName = ''
      break
    case 'list':
      forms.list.listId = ''
      break
    case 'following':
      forms.following.screenName = ''
      break
    case 'mark':
      forms.mark.users = ''
      forms.mark.lists = ''
      forms.mark.followingNames = ''
      forms.mark.timestamp = ''
      break
    case 'batch':
      forms.batch.users = ''
      forms.batch.lists = ''
      forms.batch.followingNames = ''
      break
    case 'jsonfile':
      forms.jsonfile.paths = ''
      break
    case 'jsonfolder':
      forms.jsonfolder.paths = ''
      break
  }
}

// Load initial data
onMounted(async () => {
  try {
    await taskStore.fetchTasks()
  } catch (error) {
    console.error('Failed to load tasks:', error)
  }
})
</script>

<style scoped lang="scss">
.tasks-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.task-tabs {
  display: flex;
  gap: var(--space-2);
  overflow-x: auto;
  padding-bottom: var(--space-2);

  &::-webkit-scrollbar {
    height: 4px;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--bg-elevated);
    border-radius: 2px;
  }
}

.tab-btn {
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid transparent;
  white-space: nowrap;
  transition: all var(--duration-fast);

  &:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }

  &.active {
    background: var(--accent-primary);
    color: white;
    border-color: var(--accent-primary);
  }
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
  flex-wrap: wrap;
  gap: var(--space-3);
}

.card-title {
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
}

.card-body {
  padding: var(--space-5);
}

.form-card .card-body {
  max-height: 500px;
  overflow-y: auto;
}

.task-form {
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
.form-select,
.form-textarea {
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

.form-textarea {
  resize: vertical;
  min-height: 80px;
  font-family: var(--font-mono);
  font-size: var(--text-sm);
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

.form-hint {
  font-size: var(--text-xs);
  color: var(--text-tertiary);
  line-height: 1.4;
}

.form-actions {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
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

// Task list section
.tasks-toolbar {
  display: flex;
  gap: var(--space-3);
  align-items: center;
}

.search-input {
  width: 240px;
}

.form-select {
  cursor: pointer;
  padding-right: 32px;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%238b949e'%3E%3Cpath d='M6 9L1 4h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
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

.task-list {
  display: flex;
  flex-direction: column;
}

@media (max-width: 767px) {
  .task-tabs {
    flex-wrap: nowrap;
    -webkit-overflow-scrolling: touch;
  }

  .tab-btn {
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-xs);
  }

  .tasks-toolbar {
    flex-direction: column;
    width: 100%;
  }

  .search-input,
  .form-select {
    width: 100%;
  }

  .form-actions {
    flex-direction: column;
  }

  .form-actions .btn {
    width: 100%;
  }
}
</style>
