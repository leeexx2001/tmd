// Database Store - Data Management State
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, List, UserEntity, ListEntity, UserLink, PaginationData } from '@/types'
import { databaseApi } from '@/api/database'

export const useDBStore = defineStore('database', () => {
  // State - 5 data tables
  const users = ref<PaginationData<User>>({ data: [], total: 0, page: 1, pageSize: 200, totalPages: 1 })
  const lists = ref<PaginationData<List>>({ data: [], total: 0, page: 1, pageSize: 200, totalPages: 1 })
  const entities = ref<PaginationData<UserEntity>>({ data: [], total: 0, page: 1, pageSize: 200, totalPages: 1 })
  const listEntities = ref<PaginationData<ListEntity>>({ data: [], total: 0, page: 1, pageSize: 200, totalPages: 1 })
  const userLinks = ref<PaginationData<UserLink>>({ data: [], total: 0, page: 1, pageSize: 200, totalPages: 1 })

  const currentTable = ref<'users' | 'lists' | 'entities' | 'listEntities' | 'userLinks'>('users')
  const sort = ref({ sortBy: 'id', sortOrder: 'desc' as 'asc' | 'desc' })
  const search = ref('')
  const loading = ref(false)

  // Getters
  const currentData = computed(() => {
    switch (currentTable.value) {
      case 'users': return users.value
      case 'lists': return lists.value
      case 'entities': return entities.value
      case 'listEntities': return listEntities.value
      case 'userLinks': return userLinks.value
      default: return users.value
    }
  })

  const tableLabels: Record<string, string> = {
    users: '用户',
    lists: '列表',
    entities: '实体',
    listEntities: '列表实体',
    userLinks: '用户链接'
  }

  // Actions
  async function fetchData(table?: typeof currentTable.value) {
    const targetTable = table || currentTable.value
    loading.value = true

    try {
      let result
      switch (targetTable) {
        case 'users':
          result = await databaseApi.getUsers(users.value.page)
          users.value = result
          break
        case 'lists':
          result = await databaseApi.getLists(lists.value.page)
          lists.value = result
          break
        case 'entities':
          result = await databaseApi.getEntities(entities.value.page)
          entities.value = result
          break
        case 'listEntities':
          result = await databaseApi.getListEntities(listEntities.value.page)
          listEntities.value = result
          break
        case 'userLinks':
          result = await databaseApi.getUserLinks(userLinks.value.page)
          userLinks.value = result
          break
      }
    } catch (error) {
      console.error(`Failed to fetch ${targetTable}:`, error)
      throw error
    } finally {
      loading.value = false
    }
  }

  function changePage(delta: number) {
    const tableMap = {
      users: users,
      lists: lists,
      entities: entities,
      listEntities: listEntities,
      userLinks: userLinks
    }

    const current = tableMap[currentTable.value]
    const newPage = Math.max(1, Math.min(current.value.totalPages, current.value.page + delta))
    current.value.page = newPage

    fetchData()
  }

  function toggleSort(field: string) {
    if (sort.value.sortBy === field) {
      sort.value.sortOrder = sort.value.sortOrder === 'asc' ? 'desc' : 'asc'
    } else {
      sort.value.sortBy = field
      sort.value.sortOrder = 'asc'
    }

    fetchData()
  }

  async function editItem(table: string, id: number, data: any) {
    loading.value = true
    try {
      switch (table) {
        case 'users':
          await databaseApi.updateUser(id, data)
          break
        case 'lists':
          await databaseApi.updateList(id, data)
          break
        case 'entities':
          await databaseApi.updateEntity(id, data)
          break
        case 'listEntities':
          await databaseApi.updateListEntity(id, data)
          break
        case 'userLinks':
          await databaseApi.updateUserLink(id, data)
          break
      }

      await fetchData(table as any)
    } catch (error) {
      console.error(`Failed to update ${table} item ${id}:`, error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function deleteItem(table: string, id: number) {
    loading.value = true
    try {
      switch (table) {
        case 'users':
          await databaseApi.deleteUser(id)
          break
        case 'lists':
          await databaseApi.deleteList(id)
          break
        case 'entities':
          await databaseApi.deleteEntity(id)
          break
        case 'listEntities':
          await databaseApi.deleteListEntity(id)
          break
        case 'userLinks':
          await databaseApi.deleteUserLink(id)
          break
      }

      await fetchData(table as any)
    } catch (error) {
      console.error(`Failed to delete ${table} item ${id}:`, error)
      throw error
    } finally {
      loading.value = false
    }
  }

  function setCurrentTable(table: typeof currentTable.value) {
    if (currentTable.value !== table) {
      currentTable.value = table
      sort.value = { sortBy: 'id', sortOrder: 'desc' }
      search.value = ''
      fetchData(table)
    }
  }

  return {
    users,
    lists,
    entities,
    listEntities,
    userLinks,
    currentTable,
    sort,
    search,
    loading,
    currentData,
    tableLabels,
    fetchData,
    changePage,
    toggleSort,
    editItem,
    deleteItem,
    setCurrentTable
  }
})
