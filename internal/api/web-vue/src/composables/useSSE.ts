// useSSE - Server-Sent Events Composable with Enhanced Error Handling
import { onMounted, onUnmounted, computed } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { useAppStore } from '@/stores/appStore'
import { toast } from '@/composables/useToast'

export function useSSE() {
  const taskStore = useTaskStore()
  const appStore = useAppStore()

  let eventSource: EventSource | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectAttempts = 0
  let manualDisconnect = false

  const baseReconnectDelay = 2000
  const maxReconnectDelay = 30000
  const maxReconnectAttempts = 20

  const connectionStatus = computed(() => {
    if (!eventSource) return 'disconnected'
    if (eventSource.readyState === EventSource.CONNECTING) return 'connecting'
    if (eventSource.readyState === EventSource.OPEN) return 'connected'
    return 'disconnected'
  })

  const isConnecting = computed(() => connectionStatus.value === 'connecting')
  const isConnected = computed(() => connectionStatus.value === 'connected')

  function connect() {
    if (eventSource || manualDisconnect) return

    // Check if we've exceeded max reconnect attempts
    if (reconnectAttempts >= maxReconnectAttempts) {
      console.warn(`[SSE] 已达到最大重连次数 (${maxReconnectAttempts})，停止自动重连`)
      toast.error('实时连接已断开，请刷新页面重试', { duration: 0 })
      return
    }

    try {
      eventSource = new EventSource('/api/v1/sse/tasks')

      eventSource.onopen = () => {
        console.log('[SSE] 连接已建立')
        appStore.setSSEConnected(true)
        reconnectAttempts = 0

        // Reconnected after disconnection - refresh data
        if (reconnectAttempts > 0 || taskStore.tasks.length === 0) {
          taskStore.fetchTasks().catch(err => {
            console.warn('[SSE] 重连后刷新任务失败:', err)
          })
        }
      }

      eventSource.addEventListener('tasks', (e) => {
        try {
          const data = JSON.parse(e.data)
          const tasks = data.tasks || data || []
          
          if (Array.isArray(tasks)) {
            taskStore.setTasks(tasks)
          }
        } catch (err) {
          console.warn('[SSE] tasks 解析错误:', err)
        }
      })

      eventSource.addEventListener('schedules', (e) => {
        try {
          const data = JSON.parse(e.data)
          console.log('[SSE] schedules 更新:', data.entries?.length || 0, '条记录')
          // Will be handled by scheduleStore when implemented
        } catch (err) {
          console.warn('[SSE] schedules 解析错误:', err)
        }
      })

      eventSource.addEventListener('notification', (e) => {
        try {
          const notif = JSON.parse(e.data)
          const type = notif.type === 'task_completed' ? 'success' :
                       notif.type === 'task_failed' ? 'error' :
                       notif.type === 'task_cancelled' ? 'warning' :
                       notif.type === 'schedule_warning' ? 'warning' : 'info'

          toast[type](notif.message || '通知', {
            duration: notif.type === 'task_failed' ? 6000 : 4000
          })
        } catch (err) {
          console.warn('[SSE] notification 解析错误:', err)
        }
      })

      eventSource.addEventListener('server_shutdown', (e) => {
        try {
          const data = JSON.parse(e.data)
          handleServerShutdown(data.message || '服务器正在关闭')
        } catch (err) {
          handleServerShutdown('服务器正在关闭')
        }
      })

      eventSource.onerror = (event) => {
        console.error('[SSE] 连接错误:', event)
        handleError()
      }

    } catch (error) {
      console.error('[SSE] 创建EventSource失败:', error)
      handleError()
    }
  }

  function handleError() {
    // Close existing connection
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }

    appStore.setSSEConnected(false)

    // Check if server is still alive after several failed attempts
    if (reconnectAttempts >= 5 && reconnectAttempts % 3 === 0) {
      checkServerHealth()
    }

    // Don't reconnect if manually disconnected or exceeded max attempts
    if (manualDisconnect || reconnectAttempts >= maxReconnectAttempts) {
      return
    }

    // Calculate delay with exponential backoff
    const delay = Math.min(
      baseReconnectDelay * Math.pow(2, reconnectAttempts),
      maxReconnectDelay
    )

    reconnectAttempts++
    
    // Show warning after multiple failed attempts
    if (reconnectAttempts === 3) {
      toast.warning('实时连接不稳定，正在尝试重新连接...')
    } else if (reconnectAttempts === 10) {
      toast.error('实时连接持续失败，部分功能可能受限')
    }

    console.warn(`[SSE] 连接断开，${(delay / 1000).toFixed(1)}s 后重试（第 ${reconnectAttempts}/${maxReconnectAttempts} 次）`)

    reconnectTimer = setTimeout(() => {
      connect()
    }, delay)
  }

  async function checkServerHealth() {
    try {
      const response = await fetch('/api/v1/health', {
        method: 'GET',
        headers: { 'Accept': 'application/json' },
        signal: AbortSignal.timeout(5000) // 5 second timeout
      })
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }
      
      console.log('[SSE] 服务器健康检查通过')
      
    } catch (error) {
      console.error('[SSE] 服务器健康检查失败:', error)
      
      if (reconnectAttempts >= 10) {
        handleServerShutdown('无法连接到服务器')
      }
    }
  }

  function disconnect() {
    manualDisconnect = true
    reconnectAttempts = 0

    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }

    if (eventSource) {
      eventSource.close()
      eventSource = null
    }

    appStore.setSSEConnected(false)
    console.log('[SSE] 已手动断开连接')
  }

  function reconnect() {
    // Reset state and attempt to reconnect
    manualDisconnect = false
    reconnectAttempts = 0
    
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }

    if (eventSource) {
      eventSource.close()
      eventSource = null
    }

    console.log('[SSE] 手动触发重新连接')
    connect()
  }

  function handleServerShutdown(message: string) {
    disconnect()
    toast.error(message, { duration: 0 })
    console.error('[SSE] 服务器已关闭:', message)
  }

  // Lifecycle hooks
  onMounted(() => {
    connect()
  })

  onUnmounted(() => {
    disconnect()
  })

  return {
    connect,
    disconnect,
    reconnect,
    connectionStatus,
    isConnecting,
    isConnected
  }
}
