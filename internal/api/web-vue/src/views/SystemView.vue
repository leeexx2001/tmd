<template>
  <div class="system-page">
    <!-- System Tabs -->
    <div class="system-tabs">
      <button
        v-for="tab in systemTabs"
        :key="tab.id"
        class="tab-btn"
        :class="{ active: activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        {{ tab.icon }} {{ tab.label }}
      </button>
    </div>

    <!-- Config Panel -->
    <div v-if="activeTab === 'config'" class="card">
      <div class="card-header">
        <div class="card-title">⚙️ 系统配置</div>
        <div class="config-actions">
          <button
            class="btn btn-ghost btn-sm"
            :class="{ active: configMode === 'form' }"
            @click="configMode = 'form'"
          >
            📝 表单模式
          </button>
          <button
            class="btn btn-ghost btn-sm"
            :class="{ active: configMode === 'yaml' }"
            @click="configMode = 'yaml'"
          >
            📄 YAML 模式
          </button>
        </div>
      </div>
      <div class="card-body">
        <!-- Form Mode -->
        <div v-if="configMode === 'form'" class="config-form">
          <div v-if="configFields.length === 0" class="empty-state">
            <span>加载配置字段中...</span>
          </div>
          <div v-else class="fields-grid">
            <div
              v-for="field in configFields"
              :key="field.name"
              class="field-item"
            >
              <label class="field-label">{{ field.label }}</label>
              <input
                v-model="formData[field.name]"
                :type="field.type"
                class="form-input"
                :placeholder="field.placeholder || ''"
              />
              <p v-if="field.prompt" class="field-hint">{{ field.prompt }}</p>
            </div>
          </div>

          <div class="form-actions">
            <button
              class="btn btn-primary"
              @click="handleSaveConfigForm"
              :disabled="saveLoading"
            >
              {{ saveLoading ? '保存中...' : '💾 保存配置' }}
            </button>
          </div>
        </div>

        <!-- YAML Mode with CodeMirror -->
        <div v-else class="yaml-editor-container">
          <div ref="codeMirrorContainer" class="codemirror-container"></div>

          <div class="editor-actions">
            <button
              class="btn btn-primary"
              @click="handleSaveConfigYaml"
              :disabled="saveLoading"
            >
              {{ saveLoading ? '保存中...' : '💾 保存 YAML' }}
            </button>
            <div class="editor-hint">
              <span>⚠️ 修改配置后可能需要重启服务生效</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Cookies Panel -->
    <div v-else-if="activeTab === 'cookies'" class="card">
      <div class="card-header">
        <div class="card-title">🍪 Cookie 管理</div>
        <button class="btn btn-primary btn-sm" @click="showAddCookieModal = true">
          ➕ 添加 Cookie
        </button>
      </div>
      <div class="card-body">
        <div v-if="cookies.length === 0" class="empty-state">
          <div class="empty-icon">🍪</div>
          <div class="empty-title">暂无 Cookie</div>
          <div class="empty-desc">添加 Twitter 账号的 Cookie 以启用 API 访问</div>
        </div>
        <div v-else class="cookie-list">
          <div
            v-for="(cookie, index) in cookies"
            :key="index"
            class="cookie-item"
          >
            <div class="cookie-info">
              <div class="cookie-label">Cookie #{{ index + 1 }}</div>
              <div class="cookie-details">
                <code>auth_token: {{ cookie.auth_token?.substring(0, 20) }}...</code>
                <code>ct0: {{ cookie.ct0?.substring(0, 10) }}...</code>
              </div>
            </div>
            <button
              class="btn btn-danger btn-sm"
              @click="handleDeleteCookie(index)"
            >
              🗑️ 删除
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Logs Panel -->
    <div v-else-if="activeTab === 'logs'" class="card">
      <div class="card-header">
        <div class="card-title">📋 日志查看器</div>
        <div class="log-toolbar">
          <select v-model="logLevel" class="form-select form-select-sm">
            <option value="">所有级别</option>
            <option value="DEBUG">DEBUG</option>
            <option value="INFO">INFO</option>
            <option value="WARNING">WARNING</option>
            <option value="ERROR">ERROR</option>
          </select>
          <button class="btn btn-ghost btn-sm" @click="fetchLogs">
            🔄 刷新
          </button>
        </div>
      </div>
      <div class="card-body log-container">
        <div v-if="logs.length === 0" class="empty-state">
          <div class="empty-icon">📋</div>
          <div class="empty-title">暂无日志</div>
        </div>
        <pre v-else class="log-content"><code>{{ logs.join('\n') }}</code></pre>
      </div>
    </div>

    <!-- Add Cookie Modal (Drawer) -->
    <Drawer
      v-model="showAddCookieModal"
      title="添加 Cookie"
      footer="true"
    >
      <form @submit.prevent="handleAddCookie" class="edit-form">
        <div class="form-group">
          <label class="form-label">auth_token</label>
          <input
            v-model="newCookie.auth_token"
            type="text"
            class="form-input"
            placeholder="输入 auth_token 值"
            required
          />
        </div>
        <div class="form-group">
          <label class="form-label">ct0</label>
          <input
            v-model="newCookie.ct0"
            type="text"
            class="form-input"
            placeholder="输入 ct0 值"
            required
          />
        </div>
        <div class="form-actions">
          <button type="submit" class="btn btn-primary">添加</button>
          <button type="button" class="btn btn-secondary" @click="showAddCookieModal = false">
            取消
          </button>
        </div>
      </form>
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, nextTick } from 'vue'
import { configApi, logsApi } from '@/api/config'
import { toast } from '@/composables/useToast'
import { useCodeMirror } from '@/composables/useCodeMirror'
import Drawer from '@/components/layout/Drawer.vue'
import type { ConfigField, CookieItem } from '@/types'

// System tabs
const systemTabs = [
  { id: 'config', label: '配置', icon: '⚙️' },
  { id: 'cookies', label: 'Cookie', icon: '🍪' },
  { id: 'logs', label: '日志', icon: '📋' }
]

const activeTab = ref('config')

// Config state
const configMode = ref<'form' | 'yaml'>('yaml')
const rawConfig = ref('')
const configFields = ref<ConfigField[]>([])
const formData = reactive<Record<string, string | number | boolean>>({})
const saveLoading = ref(false)
const codeMirrorContainer = ref<HTMLElement | null>(null)

// Initialize CodeMirror editor
const { containerRef: cmRef } = useCodeMirror({
  content: rawConfig,
  mode: 'yaml',
  theme: 'material-darker',
  lineNumbers: true,
  readOnly: false
})

// Cookies state
const cookies = ref<CookieItem[]>([])
const showAddCookieModal = ref(false)
const newCookie = reactive({
  auth_token: '',
  ct0: ''
})

// Logs state
const logs = ref<string[]>([])
const logLevel = ref('')

// Load initial data
onMounted(async () => {
  try {
    await Promise.all([
      fetchConfig(),
      fetchCookies(),
      fetchLogs()
    ])
    
    // Initialize CodeMirror after config is loaded
    await nextTick()
    if (codeMirrorContainer.value) {
      cmRef.value = codeMirrorContainer.value
    }
  } catch (error) {
    console.error('Failed to load system data:', error)
  }
})

async function fetchConfig() {
  try {
    const [raw, fields] = await Promise.all([
      configApi.getConfig(),
      configApi.getConfigFields()
    ])

    rawConfig.value = raw || ''
    configFields.value = fields || []

    // Initialize form data with current values
    fields.forEach(field => {
      formData[field.name] = field.value || ''
    })
  } catch (error) {
    console.error('Failed to load config:', error)
  }
}

async function handleSaveConfigYaml() {
  if (!rawConfig.value.trim()) {
    toast.warning('配置内容不能为空')
    return
  }

  saveLoading.value = true

  try {
    await configApi.updateConfigRaw(rawConfig.value)
    toast.success('配置已保存，可能需要重启服务生效')
  } catch (error: any) {
    toast.error(error.message || '保存失败')
  } finally {
    saveLoading.value = false
  }
}

async function handleSaveConfigForm() {
  saveLoading.value = true

  try {
    // Convert form data to YAML format (simplified)
    const yamlLines: string[] = []
    configFields.value.forEach(field => {
      const value = formData[field.name]
      yamlLines.push(`${field.name}: ${value}`)
    })

    await configApi.updateConfigRaw(yamlLines.join('\n'))
    toast.success('配置已保存，可能需要重启服务生效')
  } catch (error: any) {
    toast.error(error.message || '保存失败')
  } finally {
    saveLoading.value = false
  }
}

async function fetchCookies() {
  try {
    cookies.value = await configApi.getCookies()
  } catch (error) {
    console.error('Failed to load cookies:', error)
  }
}

async function handleAddCookie() {
  if (!newCookie.auth_token.trim() || !newCookie.ct0.trim()) {
    toast.warning('请填写完整的 Cookie 信息')
    return
  }

  try {
    await configApi.addCookie({ ...newCookie })
    toast.success('Cookie 已添加')

    newCookie.auth_token = ''
    newCookie.ct0 = ''
    showAddCookieModal.value = false

    await fetchCookies()
  } catch (error: any) {
    toast.error(error.message || '添加失败')
  }
}

async function handleDeleteCookie(index: number) {
  if (!confirm('确定要删除这个 Cookie 吗？')) return

  try {
    await configApi.deleteCookie(index)
    toast.success('Cookie 已删除')
    await fetchCookies()
  } catch (error: any) {
    toast.error(error.message || '删除失败')
  }
}

async function fetchLogs() {
  try {
    const response = await logsApi.getLogs()
    // Assuming response is an array of log strings or object
    logs.value = Array.isArray(response) ? response : [JSON.stringify(response, null, 2)]
  } catch (error) {
    console.error('Failed to load logs:', error)
  }
}
</script>

<style scoped lang="scss">
.system-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.system-tabs {
  display: flex;
  gap: var(--space-2);
}

.tab-btn {
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid transparent;
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

.config-actions,
.log-toolbar {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}

.config-actions .btn.active {
  background: var(--bg-tertiary);
  color: var(--accent-primary);
}

.form-select {
  cursor: pointer;
  padding-right: 32px;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%238b949e'%3E%3Cpath d='M6 9L1 4h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
}

.form-select-sm {
  padding: 6px 32px 6px 10px;
  font-size: var(--text-sm);
}

// Config Form Styles
.fields-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-4);
}

.field-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.field-label {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--text-secondary);
}

.form-input,
.yaml-editor {
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

.field-hint {
  font-size: var(--text-xs);
  color: var(--text-tertiary);
  line-height: 1.4;
}

.form-actions {
  display: flex;
  gap: var(--space-3);
  justify-content: flex-end;
  margin-top: var(--space-5);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border-primary);
}

// YAML Editor
.yaml-editor-container {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.codemirror-container {
  min-height: 400px;
  max-height: 600px;
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  overflow: hidden;
  
  :deep(.CodeMirror) {
    height: auto !important;
    font-family: var(--font-mono) !important;
    font-size: 13px !important;
    line-height: 1.6 !important;
  }
}

.yaml-editor {
  min-height: 400px;
  max-height: 600px;
  font-family: var(--font-mono);
  font-size: var(--text-sm);
  line-height: 1.6;
  resize: vertical;
}

.editor-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: var(--space-4);
}

.editor-hint {
  font-size: var(--text-xs);
  color: var(--warning);
}

// Cookies List
.cookie-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.cookie-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4);
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  gap: var(--space-4);
}

.cookie-info {
  flex: 1;
  min-width: 0;
}

.cookie-label {
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin-bottom: var(--space-2);
}

.cookie-details {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);

  code {
    font-family: var(--font-mono);
    font-size: var(--text-xs);
    color: var(--text-secondary);
    background: var(--bg-primary);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
  }
}

// Log Viewer
.log-container {
  min-height: 400px;
  max-height: 600px;
  overflow: auto;
}

.log-content {
  margin: 0;
  padding: var(--space-4);
  background: var(--bg-primary);
  border-radius: var(--radius-md);
  font-family: var(--font-mono);
  font-size: var(--text-sm);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text-secondary);
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

// Empty states
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

// Edit form styles (for cookie modal)
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

@media (max-width: 767px) {
  .system-tabs {
    flex-wrap: wrap;
  }

  .tab-btn {
    flex: 1;
    text-align: center;
  }

  .fields-grid {
    grid-template-columns: 1fr;
  }

  .cookie-item {
    flex-direction: column;
    align-items: stretch;
  }

  .config-actions,
  .log-toolbar {
    flex-direction: column;
    width: 100%;
  }

  .form-actions {
    flex-direction: column-reverse;
  }

  .form-actions .btn {
    width: 100%;
  }
}
</style>
