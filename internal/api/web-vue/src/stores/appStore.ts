// App Store - Global Application State
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAppStore = defineStore('app', () => {
  // State
  const sidebarOpen = ref(false)
  const isMobile = ref(window.innerWidth < 768)
  const sseConnected = ref(false)

  // Getters
  const pageTitle = computed(() => {
    const titles: Record<string, string> = {
      overview: '概览',
      tasks: '任务中心',
      data: '数据管理',
      schedules: '定时任务',
      system: '系统'
    }
    return titles['overview'] // Will be updated by router
  })

  // Actions
  function toggleSidebar() {
    sidebarOpen.value = !sidebarOpen.value
  }

  function closeSidebar() {
    sidebarOpen.value = false
  }

  function updateMobile() {
    isMobile.value = window.innerWidth < 768
  }

  function setSSEConnected(connected: boolean) {
    sseConnected.value = connected
  }

  return {
    sidebarOpen,
    isMobile,
    sseConnected,
    pageTitle,
    toggleSidebar,
    closeSidebar,
    updateMobile,
    setSSEConnected
  }
})
