// Tasks API
import api from './client'
import type { Task, Health } from '@/types'

export const tasksApi = {
  getHealth() {
    return api.get<Health>('/api/v1/health')
  },

  getTasks() {
    return api.get<{ tasks: Task[] }>('/api/v1/tasks')
  },

  getTask(id: string) {
    return api.get<Task>(`/api/v1/tasks/${id}`)
  },

  cancelTask(id: string) {
    return api.post(`/api/v1/tasks/${id}/cancel`, {})
  },

  createUserDownload(screenName: string, opts: any) {
    return api.post(`/api/v1/users/${encodeURIComponent(screenName)}/download`, opts)
  },

  createProfileDownload(screenName: string) {
    return api.post(`/api/v1/users/${encodeURIComponent(screenName)}/profile`, {})
  },

  createUserMark(screenName: string, timestamp?: string) {
    return api.post(`/api/v1/users/${encodeURIComponent(screenName)}/mark`, timestamp ? { timestamp } : {})
  },

  createFollowingDownload(screenName: string, opts: any) {
    return api.post(`/api/v1/users/${encodeURIComponent(screenName)}/following/download`, opts)
  },

  createFollowingMark(screenName: string, timestamp?: string) {
    return api.post(`/api/v1/users/${encodeURIComponent(screenName)}/following/mark`, timestamp ? { timestamp } : {})
  },

  createListDownload(listId: number | string, opts: any) {
    return api.post(`/api/v1/lists/${encodeURIComponent(listId)}/download`, opts)
  },

  createListProfile(listId: number | string) {
    return api.post(`/api/v1/lists/${encodeURIComponent(listId)}/profile`, {})
  },

  createListMark(listId: number | string, timestamp?: string) {
    return api.post(`/api/v1/lists/${encodeURIComponent(listId)}/mark`, timestamp ? { timestamp } : {})
  },

  createBatchDownload(data: any) {
    return api.post('/api/v1/batch/download', data)
  },

  createJsonFileDownload(data: any) {
    return api.post('/api/v1/json/file/download', data)
  },

  createJsonFolderDownload(data: any) {
    return api.post('/api/v1/json/folder/download', data)
  }
}

export default tasksApi
