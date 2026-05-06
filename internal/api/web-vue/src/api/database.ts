// Database API - CRUD Operations
import api from './client'
import type { User, List, UserEntity, ListEntity, UserLink, PaginationData } from '@/types'

export const databaseApi = {
  // Users
  getUsers(page = 1, pageSize = 200) {
    return api.get<PaginationData<User>>(`/api/v1/db/users?page=${page}&pageSize=${pageSize}`)
  },

  getUser(id: number) {
    return api.get<User>(`/api/v1/db/users/${id}`)
  },

  updateUser(id: number, data: Partial<User>) {
    return api.put(`/api/v1/db/users/${id}`, data)
  },

  deleteUser(id: number) {
    return api.delete(`/api/v1/db/users/${id}`)
  },

  // Lists
  getLists(page = 1, pageSize = 200) {
    return api.get<PaginationData<List>>(`/api/v1/db/lists?page=${page}&pageSize=${pageSize}`)
  },

  updateList(id: number, data: Partial<List>) {
    return api.put(`/api/v1/db/lists/${id}`, data)
  },

  deleteList(id: number) {
    return api.delete(`/api/v1/db/lists/${id}`)
  },

  // Entities
  getEntities(page = 1, pageSize = 200) {
    return api.get<PaginationData<UserEntity>>(`/api/v1/db/entities?page=${page}&pageSize=${pageSize}`)
  },

  updateEntity(id: number, data: Partial<UserEntity>) {
    return api.put(`/api/v1/db/entities/${id}`, data)
  },

  deleteEntity(id: number) {
    return api.delete(`/api/v1/db/entities/${id}`)
  },

  // List Entities
  getListEntities(page = 1, pageSize = 200) {
    return api.get<PaginationData<ListEntity>>(`/api/v1/db/list-entities?page=${page}&pageSize=${pageSize}`)
  },

  updateListEntity(id: number, data: Partial<ListEntity>) {
    return api.put(`/api/v1/db/list-entities/${id}`, data)
  },

  deleteListEntity(id: number) {
    return api.delete(`/api/v1/db/list-entities/${id}`)
  },

  // User Links
  getUserLinks(page = 1, pageSize = 200) {
    return api.get<PaginationData<UserLink>>(`/api/v1/db/user-links?page=${page}&pageSize=${pageSize}`)
  },

  updateUserLink(id: number, data: Partial<UserLink>) {
    return api.put(`/api/v1/db/user-links/${id}`, data)
  },

  deleteUserLink(id: number) {
    return api.delete(`/api/v1/db/user-links/${id}`)
  }
}

export default databaseApi
