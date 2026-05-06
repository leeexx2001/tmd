<template>
  <div class="app-layout">
    <Sidebar :current-page="currentPage" @navigate="handleNavigate" />
    <main class="main-content">
      <Header
        :title="pageTitle"
        :sse-connected="appStore.sseConnected"
        @refresh="handleRefresh"
        @toggle-sidebar="appStore.toggleSidebar"
      />
      <div class="content-container">
        <router-view />
      </div>
    </main>
    <MobileNav :current-page="currentPage" @navigate="handleNavigate" />
    <Drawer v-model="drawerVisible" :title="drawerTitle" :footer="drawerFooter">
      <component :is="drawerContent" />
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, provide } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Sidebar from './Sidebar.vue'
import Header from './Header.vue'
import MobileNav from './MobileNav.vue'
import Drawer from './Drawer.vue'
import { useAppStore } from '@/stores/appStore'
import { useSSE } from '@/composables/useSSE'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

// Initialize SSE connection
useSSE()

// Current page detection
const currentPage = computed(() => {
  const path = route.path
  if (path === '/' || path === '') return 'overview'
  return path.replace('/', '')
})

const pageTitle = computed(() => {
  const titles: Record<string, string> = {
    overview: '概览',
    tasks: '任务中心',
    data: '数据管理',
    schedules: '定时任务',
    system: '系统'
  }
  return titles[currentPage.value] || 'TMD Pro'
})

// Drawer state (provide for child components)
const drawerVisible = ref(false)
const drawerTitle = ref('')
const drawerContent = ref<any>(null)
const drawerFooter = ref('')

provide('drawer', {
  visible: drawerVisible,
  title: drawerTitle,
  content: drawerContent,
  footer: drawerFooter,
  open: (title: string, content: any, footer = '') => {
    drawerTitle.value = title
    drawerContent.value = content
    drawerFooter.value = footer
    drawerVisible.value = true
  },
  close: () => {
    drawerVisible.value = false
  }
})

// Event handlers
function handleNavigate(page: string) {
  router.push(page === 'overview' ? '/' : `/${page}`)
}

function handleRefresh() {
  window.location.reload()
}
</script>

<style scoped lang="scss">
.app-layout {
  display: flex;
  min-height: 100vh;
}

.main-content {
  flex: 1;
  margin-left: var(--sidebar-width);
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.content-container {
  flex: 1;
  padding: var(--space-6);
  max-width: var(--content-max-width);
  width: 100%;
  margin: 0 auto;
}

@media (max-width: 1023px) {
  .main-content {
    margin-left: 0;
  }
}

@media (max-width: 767px) {
  .content-container {
    padding: var(--space-4);
    padding-bottom: calc(var(--mobile-nav-height) + var(--space-4));
  }
}
</style>
