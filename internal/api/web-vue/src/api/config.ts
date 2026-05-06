// Config & Logs & Schedules API
import api from './client'
import type { ConfigField, CookieItem, ScheduleStatus } from '@/types'

export const configApi = {
  // Configuration
  getConfig() {
    return api.get<string>('/api/v1/config/raw')
  },

  getConfigFields() {
    return api.get<ConfigField[]>('/api/v1/config/fields')
  },

  updateConfigRaw(config: string) {
    return api.put('/api/v1/config/raw', { config })
  },

  // Cookies
  getCookies() {
    return api.get<CookieItem[]>('/api/v1/cookies')
  },

  addCookie(cookie: CookieItem) {
    return api.post('/api/v1/cookies', cookie)
  },

  deleteCookie(index: number) {
    return api.delete(`/api/v1/cookies/${index}`)
  }
}

export const logsApi = {
  getLogs(page = 1) {
    return api.get(`/api/v1/logs?page=${page}`)
  }
}

export const schedulesApi = {
  getSchedules() {
    return api.get<ScheduleStatus[]>('/api/v1/schedules')
  },

  createSchedule(data: any) {
    return api.post('/api/v1/schedules', data)
  },

  updateSchedule(id: string, data: any) {
    return api.put(`/api/v1/schedules/${id}`, data)
  },

  deleteSchedule(id: string) {
    return api.delete(`/api/v1/schedules/${id}`)
  },

  toggleSchedule(id: string, enabled: boolean) {
    return api.put(`/api/v1/schedules/${id}/toggle`, { enabled })
  },

  triggerSchedule(id: string) {
    return api.post(`/api/v1/schedules/${id}/trigger`, {})
  }
}

export default configApi
