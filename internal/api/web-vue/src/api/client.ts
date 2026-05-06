// API Client - Enhanced HTTP Request Wrapper with Error Handling
import type { ApiResponse } from '@/types'
import { toast } from '@/composables/useToast'

interface RequestConfig {
  method: string
  path: string
  body?: any
  options?: RequestInit
  showError?: boolean
  showLoading?: boolean
}

class ApiClient {
  private base = ''
  private abortController: AbortController | null = null
  private requestCount = 0

  private getSignal(): AbortSignal {
    if (!this.abortController) {
      this.abortController = new AbortController()
    }
    return this.abortController.signal
  }

  abortAll() {
    if (this.abortController) {
      this.abortController.abort()
      this.abortController = null
    }
    this.requestCount = 0
  }

  private handleRequestStart(config: RequestConfig) {
    if (config.showLoading !== false) {
      this.requestCount++
      // Could show global loading indicator here
    }
  }

  private handleRequestEnd() {
    if (this.requestCount > 0) {
      this.requestCount--
      // Could hide global loading indicator here when count reaches 0
    }
  }

  private handleError(error: Error, config: RequestConfig): never {
    this.handleRequestEnd()

    // Network errors
    if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
      const networkError = new Error('网络连接失败，请检查网络设置')
      if (config.showError !== false) {
        toast.error(networkError.message)
      }
      throw networkError
    }

    // Abort errors
    if (error.name === 'AbortError') {
      const abortError = new Error('请求已取消')
      throw abortError
    }

    // Server errors with messages
    if (config.showError !== false) {
      toast.error(error.message || '请求失败')
    }

    throw error
  }

  async request<T>(method: string, path: string, body?: any, options?: RequestInit & { showError?: boolean; showLoading?: boolean }): Promise<T> {
    const config: RequestConfig = { method, path, body, options, showError: options?.showError, showLoading: options?.showLoading }
    
    this.handleRequestStart(config)

    try {
      const requestOptions: RequestInit = {
        method,
        headers: { 
          'Content-Type': 'application/json',
          'Accept': 'application/json'
        },
        signal: this.getSignal(),
        ...options
      }

      if (body && method !== 'GET') {
        requestOptions.body = JSON.stringify(body)
      }

      const res = await fetch(this.base + path, requestOptions)

      // Handle non-JSON responses
      let data: ApiResponse<T>
      try {
        data = await res.json()
      } catch (parseError) {
        throw new Error(`服务器响应解析失败 (HTTP ${res.status})`)
      }

      // Handle HTTP errors
      if (!res.ok) {
        const errorMessage = data.error || data.message || `服务器错误 (HTTP ${res.status})`
        
        // Specific status code handling
        switch (res.status) {
          case 401:
            throw new Error('未授权，请重新登录')
          case 403:
            throw new Error('没有权限执行此操作')
          case 404:
            throw new Error('请求的资源不存在')
          case 429:
            throw new Error('请求过于频繁，请稍后再试')
          case 500:
            throw new Error(errorMessage || '服务器内部错误')
          case 502:
          case 503:
          case 504:
            throw new Error('服务暂时不可用，请稍后再试')
          default:
            throw new Error(errorMessage)
        }
      }

      // Handle API-level errors
      if (!data.success) {
        throw new Error(data.error || '操作失败')
      }

      this.handleRequestEnd()
      return data.data as T

    } catch (error) {
      this.handleError(error as Error, config)
    }
  }

  get<T>(path: string, options?: RequestInit & { showError?: boolean }): Promise<T> {
    return this.request<T>('GET', path, undefined, options)
  }

  post<T>(path: string, body?: any, options?: RequestInit & { showError?: boolean }): Promise<T> {
    return this.request<T>('POST', path, body, options)
  }

  put<T>(path: string, body?: any, options?: RequestInit & { showError?: boolean }): Promise<T> {
    return this.request<T>('PUT', path, body, options)
  }

  delete<T>(path: string, options?: RequestInit & { showError?: boolean }): Promise<T> {
    return this.request<T>('DELETE', path, undefined, options)
  }

  // Utility methods for common patterns
  async getWithPagination<T>(path: string, page: number = 1, pageSize: number = 20, options?: RequestInit & { showError?: boolean }) {
    return this.get<{ data: T[]; total: number; page: number; pageSize: number; totalPages: number }>(
      `${path}?page=${page}&pageSize=${pageSize}`,
      options
    )
  }

  // Health check
  async healthCheck(): Promise<boolean> {
    try {
      await this.get('/api/v1/health', { showError: false })
      return true
    } catch {
      return false
    }
  }
}

export const api = new ApiClient()
export default api
