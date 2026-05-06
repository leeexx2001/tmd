<template>
  <div class="data-page">
    <!-- Table Tabs -->
    <div class="table-tabs">
      <button
        v-for="tab in tableTabs"
        :key="tab.id"
        class="tab-btn"
        :class="{ active: dbStore.currentTable === tab.id }"
        @click="dbStore.setCurrentTable(tab.id as any)"
      >
        {{ tab.icon }} {{ tab.label }}
        <span class="tab-count" v-if="getTableCount(tab.id) > 0">
          {{ formatNumber(getTableCount(tab.id)) }}
        </span>
      </button>
    </div>

    <!-- Data Table Card -->
    <div class="card">
      <div class="card-header">
        <div class="card-title">📊 {{ dbStore.tableLabels[dbStore.currentTable] }} 数据表</div>
        <button
          class="btn btn-ghost btn-sm"
          @click="dbStore.fetchData()"
          :disabled="dbStore.loading"
        >
          🔄 刷新
        </button>
      </div>

      <div class="card-body">
        <!-- Loading State -->
        <div v-if="dbStore.loading" class="loading-state">
          <div class="loading-spinner"></div>
          <span>加载数据中...</span>
        </div>

        <!-- Empty State -->
        <div v-else-if="dbStore.currentData.data.length === 0" class="empty-state">
          <div class="empty-icon">📊</div>
          <div class="empty-title">暂无数据</div>
          <div class="empty-desc">{{ dbStore.tableLabels[dbStore.currentTable] }} 表中还没有记录</div>
        </div>

        <!-- Desktop Table View -->
        <div v-else class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th
                  v-for="column in currentColumns"
                  :key="column.key"
                  :class="{ 'sort-active': dbStore.sort.sortBy === column.key }"
                  @click="dbStore.toggleSort(column.key)"
                >
                  {{ column.label }}
                  <span class="sort-icon">
                    {{ dbStore.sort.sortBy === column.key ? (dbStore.sort.sortOrder === 'asc' ? '↑' : '↓') : '↕' }}
                  </span>
                </th>
                <th style="width: 100px;">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in paginatedData" :key="getRowKey(item)">
                <td v-for="column in currentColumns" :key="column.key">
                  {{ formatCellValue(item, column.key) }}
                </td>
                <td>
                  <div class="action-buttons">
                    <button
                      class="btn btn-ghost btn-sm btn-icon"
                      title="编辑"
                      @click="handleEdit(item)"
                    >
                      ✏️
                    </button>
                    <button
                      class="btn btn-danger btn-sm btn-icon"
                      title="删除"
                      @click="handleDelete(item)"
                    >
                      🗑️
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>

          <!-- Pagination -->
          <Pagination
            :current-page="dbStore.currentData.page"
            :total-pages="dbStore.currentData.totalPages"
            :total="dbStore.currentData.total"
            @change="dbStore.changePage"
          />
        </div>
      </div>
    </div>

    <!-- Edit Modal (Drawer) -->
    <Drawer
      v-model="editModalVisible"
      :title="`编辑 ${dbStore.tableLabels[dbStore.currentTable]}`"
      footer="true"
    >
      <form @submit.prevent="handleSaveEdit" class="edit-form">
        <div
          v-for="field in editableFields"
          :key="field.key"
          class="form-group"
        >
          <label class="form-label">{{ field.label }}</label>
          <input
            v-if="field.type !== 'textarea'"
            v-model="editForm[field.key]"
            :type="field.type || 'text'"
            class="form-input"
          />
          <textarea
            v-else
            v-model="editForm[field.key]"
            class="form-textarea"
            rows="3"
          ></textarea>
        </div>
        <div class="form-actions">
          <button type="submit" class="btn btn-primary" :disabled="saveLoading">
            {{ saveLoading ? '保存中...' : '保存' }}
          </button>
          <button type="button" class="btn btn-secondary" @click="editModalVisible = false">
            取消
          </button>
        </div>
      </form>
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useDBStore } from '@/stores/dbStore'
import { toast } from '@/composables/useToast'
import Pagination from '@/components/business/Pagination.vue'
import Drawer from '@/components/layout/Drawer.vue'

const dbStore = useDBStore()

// Table tabs configuration
const tableTabs = [
  { id: 'users', label: '用户', icon: '👤' },
  { id: 'lists', label: '列表', icon: '📋' },
  { id: 'entities', label: '实体', icon: '📦' },
  { id: 'listEntities', label: '列表实体', icon: '🗂️' },
  { id: 'userLinks', label: '用户链接', icon: '🔗' }
]

// Column definitions for each table type
const columnDefinitions: Record<string, Array<{ key: string; label: string }>> = {
  users: [
    { key: 'id', label: 'ID' },
    { key: 'screen_name', label: 'Screen Name' },
    { key: 'name', label: '名称' },
    { key: 'protected', label: '保护' },
    { key: 'is_accessible', label: '可访问' },
    { key: 'friends_count', label: '关注数' }
  ],
  lists: [
    { key: 'id', label: 'ID' },
    { key: 'name', label: '名称' },
    { key: 'owner_user_id', label: '所有者ID' }
  ],
  entities: [
    { key: 'id', label: 'ID' },
    { key: 'user_id', label: '用户ID' },
    { key: 'name', label: '名称' },
    { key: 'latest_release_time', label: '最新发布时间' },
    { key: 'media_count', label: '媒体数量' }
  ],
  listEntities: [
    { key: 'id', label: 'ID' },
    { key: 'lst_id', label: '列表ID' },
    { key: 'name', label: '名称' },
    { key: 'parent_dir', label: '父目录' }
  ],
  userLinks: [
    { key: 'id', label: 'ID' },
    { key: 'user_id', label: '用户ID' },
    { key: 'name', label: '名称' },
    { key: 'parent_lst_entity_id', label: '父列表实体ID' }
  ]
}

// Editable fields (exclude ID)
const editableFieldsDefinitions: Record<string, Array<{ key: string; label: string; type?: string }>> = {
  users: [
    { key: 'screen_name', label: 'Screen Name' },
    { key: 'name', label: '名称' },
    { key: 'protected', label: '保护状态', type: 'checkbox' },
    { key: 'is_accessible', label: '可访问', type: 'checkbox' },
    { key: 'friends_count', label: '关注数', type: 'number' }
  ],
  lists: [
    { key: 'name', label: '名称' },
    { key: 'owner_user_id', label: '所有者ID' }
  ],
  entities: [
    { key: 'user_id', label: '用户ID' },
    { key: 'name', label: '名称' },
    { key: 'media_count', label: '媒体数量', type: 'number' }
  ],
  listEntities: [
    { key: 'lst_id', label: '列表ID', type: 'number' },
    { key: 'name', label: '名称' },
    { key: 'parent_dir', label: '父目录' }
  ],
  userLinks: [
    { key: 'user_id', label: '用户ID' },
    { key: 'name', label: '名称' },
    { key: 'parent_lst_entity_id', label: '父列表实体ID', type: 'number' }
  ]
}

// Computed properties
const currentColumns = computed(() => {
  return columnDefinitions[dbStore.currentTable] || []
})

const editableFields = computed(() => {
  return editableFieldsDefinitions[dbStore.currentTable] || []
})

const paginatedData = computed(() => {
  return dbStore.currentData.data || []
})

// Edit modal state
const editModalVisible = ref(false)
const saveLoading = ref(false)
const editingItem = ref<any>(null)
const editForm = reactive<Record<string, any>>({})

function getRowKey(item: any): number {
  return item.id
}

function formatCellValue(item: any, key: string): string {
  const value = item[key]

  if (typeof value === 'boolean') {
    return value ? '✅' : '❌'
  }

  if (value === null || value === undefined) {
    return '-'
  }

  if (key.includes('time') && typeof value === 'string') {
    try {
      return new Date(value).toLocaleString()
    } catch {
      return value
    }
  }

  return String(value)
}

function formatNumber(num: number): string {
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'k'
  }
  return String(num)
}

function getTableCount(tableId: string): number {
  switch (tableId) {
    case 'users': return dbStore.users.total
    case 'lists': return dbStore.lists.total
    case 'entities': return dbStore.entities.total
    case 'listEntities': return dbStore.listEntities.total
    case 'userLinks': return dbStore.userLinks.total
    default: return 0
  }
}

// Edit handlers
function handleEdit(item: any) {
  editingItem.value = item

  // Reset and populate form
  Object.keys(editForm).forEach(key => delete editForm[key])
  editableFields.value.forEach(field => {
    editForm[field.key] = item[field.key]
  })

  editModalVisible.value = true
}

async function handleSaveEdit() {
  if (!editingItem.value) return

  saveLoading.value = true

  try {
    await dbStore.editItem(dbStore.currentTable, editingItem.value.id, { ...editForm })
    toast.success('保存成功')
    editModalVisible.value = false
  } catch (error: any) {
    toast.error(error.message || '保存失败')
  } finally {
    saveLoading.value = false
  }
}

async function handleDelete(item: any) {
  const confirmMessage = `确定要删除这条${dbStore.tableLabels[dbStore.currentTable]}记录吗？\n\nID: ${item.id}`

  if (!confirm(confirmMessage)) return

  try {
    await dbStore.deleteItem(dbStore.currentTable, item.id)
    toast.success('删除成功')
  } catch (error: any) {
    toast.error(error.message || '删除失败')
  }
}

// Load initial data
onMounted(async () => {
  try {
    await dbStore.fetchData()
  } catch (error) {
    console.error('Failed to load data:', error)
  }
})
</script>

<style scoped lang="scss">
.data-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.table-tabs {
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
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
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

.tab-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.15);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
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

.table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;

  th, td {
    padding: var(--space-3) var(--space-4);
    text-align: left;
    border-bottom: 1px solid var(--border-primary);
    white-space: nowrap;
  }

  thead {
    background: var(--bg-tertiary);

    th {
      font-size: var(--text-sm);
      font-weight: var(--font-semibold);
      color: var(--text-secondary);
      cursor: pointer;
      user-select: none;
      transition: color var(--duration-fast);

      &:hover {
        color: var(--text-primary);
      }

      &.sort-active {
        color: var(--accent-primary);
      }
    }
  }

  tbody tr {
    transition: background var(--duration-fast);

    &:hover {
      background: var(--bg-tertiary);
    }
  }

  td {
    font-size: var(--text-sm);
    color: var(--text-primary);
  }
}

.sort-icon {
  margin-left: var(--space-1);
  opacity: 0.5;
  font-size: var(--text-xs);
}

.action-buttons {
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

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
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

.btn-sm {
  padding: 6px 10px;
  font-size: var(--text-xs);
}

.btn-icon {
  width: 28px;
  height: 28px;
  padding: 0;
  border-radius: var(--radius-sm);
}

// Edit form styles
.edit-form {
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
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
  font-family: var(--font-mono);
  font-size: var(--text-sm);
}

.form-actions {
  display: flex;
  gap: var(--space-3);
  justify-content: flex-end;
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-primary);
}

@media (max-width: 767px) {
  .table-tabs {
    flex-wrap: nowrap;
    -webkit-overflow-scrolling: touch;
  }

  .tab-btn {
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-xs);
  }

  .data-table {
    font-size: var(--text-xs);

    th, td {
      padding: var(--space-2) var(--space-3);
    }
  }

  .form-actions {
    flex-direction: column-reverse;
  }

  .form-actions .btn {
    width: 100%;
  }
}
</style>
