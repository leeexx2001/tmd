// Vue Router Configuration
import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/components/layout/AppLayout.vue'),
    children: [
      {
        path: '',
        name: 'overview',
        component: () => import('@/views/OverviewView.vue'),
        meta: { title: '概览', icon: '📊' }
      },
      {
        path: 'tasks',
        name: 'tasks',
        component: () => import('@/views/TasksView.vue'),
        meta: { title: '任务', icon: '🚀' }
      },
      {
        path: 'data',
        name: 'data',
        redirect: '/data/users',
        children: [
          {
            path: 'users',
            name: 'data-users',
            component: () => import('@/views/DataView.vue'),
            meta: { title: '数据 - 用户', icon: '👥' }
          },
          {
            path: 'lists',
            name: 'data-lists',
            component: () => import('@/views/DataView.vue'),
            meta: { title: '数据 - 列表', icon: '📋' }
          },
          {
            path: 'entities',
            name: 'data-entities',
            component: () => import('@/views/DataView.vue'),
            meta: { title: '数据 - 实体', icon: '📦' }
          },
          {
            path: 'list-entities',
            name: 'data-list-entities',
            component: () => import('@/views/DataView.vue'),
            meta: { title: '数据 - 列表实体', icon: '🗂️' }
          },
          {
            path: 'user-links',
            name: 'data-user-links',
            component: () => import('@/views/DataView.vue'),
            meta: { title: '数据 - 用户链接', icon: '🔗' }
          }
        ]
      },
      {
        path: 'schedules',
        name: 'schedules',
        component: () => import('@/views/SchedulesView.vue'),
        meta: { title: '定时任务', icon: '⏰' }
      },
      {
        path: 'system',
        name: 'system',
        redirect: '/system/config',
        children: [
          {
            path: 'config',
            name: 'system-config',
            component: () => import('@/views/SystemView.vue'),
            meta: { title: '系统 - 配置', icon: '⚙️' }
          },
          {
            path: 'cookies',
            name: 'system-cookies',
            component: () => import('@/views/SystemView.vue'),
            meta: { title: '系统 - Cookie', icon: '🍪' }
          },
          {
            path: 'logs',
            name: 'system-logs',
            component: () => import('@/views/SystemView.vue'),
            meta: { title: '系统 - 日志', icon: '📋' }
          }
        ]
      }
    ]
  },
  // 404 Not Found
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: '页面未找到' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, _from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    } else if (to.hash) {
      return { el: to.hash, behavior: 'smooth' }
    } else {
      return { top: 0 }
    }
  }
})

// Navigation guards
router.beforeEach((to, _from, next) => {
  // Update page title
  const title = to.meta.title as string
  if (title) {
    document.title = `${title} - TMD Pro`
  }
  
  next()
})

export default router
