// Task Store - Task Management State
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Task } from '@/types'
import { tasksApi } from '@/api/tasks'

export const useTaskStore = defineStore('task', () => {
  // State
  const tasks = ref<Task[]>([])
  const taskFilter = ref<'all' | 'running' | 'queued' | 'completed' | 'failed'>('all')
  const taskSearch = ref('')
  const loading = ref(false)

  // Getters
  const filteredTasks = computed(() => {
    let result = tasks.value

    // Filter by status
    if (taskFilter.value !== 'all') {
      result = result.filter(t => t.status === taskFilter.value)
    }

    // Filter by search
    if (taskSearch.value) {
      const search = taskSearch.value.toLowerCase()
      result = result.filter(t => {
        const target = t.data?.screen_name || t.data?.list_id || ''
        return target.toString().toLowerCase().includes(search) ||
               t.task_id.toLowerCase().includes(search)
      })
    }

    return result
  })

  const taskStats = computed(() => ({
    queued: tasks.value.filter(t => t.status === 'queued').length,
    running: tasks.value.filter(t => t.status === 'running').length,
    completed: tasks.value.filter(t => t.status === 'completed').length,
    failed: tasks.value.filter(t => t.status === 'failed').length,
    cancelled: tasks.value.filter(t => t.status === 'cancelled').length
  }))

  const recentTasks = computed(() => tasks.value.slice(0, 5))

  // Actions
  async function fetchTasks() {
    loading.value = true
    try {
      const data = await tasksApi.getTasks()
      tasks.value = data.tasks || []
    } catch (error) {
      console.error('Failed to fetch tasks:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function cancelTask(id: string) {
    try {
      await tasksApi.cancelTask(id)
      // Update local state optimistically
      const task = tasks.value.find(t => t.task_id === id)
      if (task) {
        task.status = 'cancelled'
      }
    } catch (error) {
      console.error('Failed to cancel task:', error)
      throw error
    }
  }

  function setTasks(newTasks: Task[]) {
    tasks.value = newTasks
  }

  function updateTaskFilter(filter: typeof taskFilter.value) {
    taskFilter.value = filter
  }

  function updateTaskSearch(search: string) {
    taskSearch.value = search
  }

  return {
    tasks,
    taskFilter,
    taskSearch,
    loading,
    filteredTasks,
    taskStats,
    recentTasks,
    fetchTasks,
    cancelTask,
    setTasks,
    updateTaskFilter,
    updateTaskSearch
  }
})
