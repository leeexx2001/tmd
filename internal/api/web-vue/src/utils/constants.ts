// Constants - Application-wide Constants

// API Endpoints
export const API_ENDPOINTS = {
  TASKS: '/api/v1/tasks',
  HEALTH: '/api/v1/health',
  SSE_TASKS: '/api/v1/sse/tasks',
  
  // Database
  DB_USERS: '/api/v1/db/users',
  DB_LISTS: '/api/v1/db/lists',
  DB_ENTITIES: '/api/v1/db/entities',
  DB_LIST_ENTITIES: '/api/v1/db/list-entities',
  DB_USER_LINKS: '/api/v1/db/user-links',
  
  // Config
  CONFIG_RAW: '/api/v1/config/raw',
  CONFIG_FIELDS: '/api/v1/config/fields',
  COOKIES: '/api/v1/cookies',
  
  // Schedules
  SCHEDULES: '/api/v1/schedules',
  
  // Logs
  LOGS: '/api/v1/logs'
} as const

// Task Types
export const TASK_TYPES = {
  LIST: 'list' as const,
  USER: 'user' as const,
  FOLLOWING: 'following' as const,
  LIKES: 'likes' as const,
  MEDIA: 'media' as const,
  BATCH: 'batch' as const,
  JSON_FILE: 'jsonfile' as const,
  JSON_FOLDER: 'jsonfolder' as const,
  MARK: 'mark' as const
} as const

export const TASK_TYPE_LABELS: Record<string, string> = {
  [TASK_TYPES.LIST]: '列表下载',
  [TASK_TYPES.USER]: '用户下载',
  [TASK_TYPES.FOLLOWING]: '关注者下载',
  [TASK_TYPES.LIKES]: '点赞下载',
  [TASK_TYPES.MEDIA]: '媒体下载',
  [TASK_TYPES.BATCH]: '批量下载',
  [TASK_TYPES.JSON_FILE]: 'JSON文件导入',
  [TASK_TYPES.JSON_FOLDER]: 'JSON文件夹导入',
  [TASK_TYPES.MARK]: '标记任务'
}

// Task Status
export const TASK_STATUS = {
  QUEUED: 'queued' as const,
  RUNNING: 'running' as const,
  COMPLETED: 'completed' as const,
  FAILED: 'failed' as const,
  CANCELLED: 'cancelled' as const
} as const

export const TASK_STATUS_LABELS: Record<string, string> = {
  [TASK_STATUS.QUEUED]: '排队中',
  [TASK_STATUS.RUNNING]: '运行中',
  [TASK_STATUS.COMPLETED]: '已完成',
  [TASK_STATUS.FAILED]: '失败',
  [TASK_STATUS.CANCELLED]: '已取消'
}

export const TASK_STATUS_COLORS: Record<string, string> = {
  [TASK_STATUS.QUEUED]: 'var(--text-tertiary)',
  [TASK_STATUS.RUNNING]: 'var(--info)',
  [TASK_STATUS.COMPLETED]: 'var(--success)',
  [TASK_STATUS.FAILED]: 'var(--danger)',
  [TASK_STATUS.CANCELLED]: 'var(--warning)'
}

// Database Table Types
export const TABLE_TYPES = {
  USERS: 'users' as const,
  LISTS: 'lists' as const,
  ENTITIES: 'entities' as const,
  LIST_ENTITIES: 'listEntities' as const,
  USER_LINKS: 'userLinks' as const
} as const

export const TABLE_LABELS: Record<string, string> = {
  [TABLE_TYPES.USERS]: '用户',
  [TABLE_TYPES.LISTS]: '列表',
  [TABLE_TYPES.ENTITIES]: '实体',
  [TABLE_TYPES.LIST_ENTITIES]: '列表实体',
  [TABLE_TYPES.USER_LINKS]: '用户链接'
}

export const TABLE_ICONS: Record<string, string> = {
  [TABLE_TYPES.USERS]: '👤',
  [TABLE_TYPES.LISTS]: '📋',
  [TABLE_TYPES.ENTITIES]: '📦',
  [TABLE_TYPES.LIST_ENTITIES]: '🗂️',
  [TABLE_TYPES.USER_LINKS]: '🔗'
}

// Pagination
export const DEFAULT_PAGE_SIZE = 200
export const PAGE_SIZE_OPTIONS = [50, 100, 200, 500]

// SSE Configuration
export const SSE_CONFIG = {
  BASE_RECONNECT_DELAY: 2000,
  MAX_RECONNECT_DELAY: 30000,
  MAX_RECONNECT_ATTEMPTS: 20,
  HEARTBEAT_INTERVAL: 30000
} as const

// Toast Duration (ms)
export const TOAST_DURATION = {
  SHORT: 3000,
  NORMAL: 4000,
  LONG: 6000,
  PERSISTENT: 0
} as const

// Animation Duration (ms)
export const ANIMATION_DURATION = {
  FAST: 150,
  NORMAL: 250,
  SLOW: 350
} as const

// Breakpoints
export const BREAKPOINTS = {
  MOBILE: 767,
  TABLET: 1023,
  DESKTOP: 1366,
  LARGE: 1920
} as const

// Local Storage Keys
export const STORAGE_KEYS = {
  THEME: 'tmd-theme',
  SIDEBAR_COLLAPSED: 'tmd-sidebar-collapsed',
  LAST_ACTIVE_TABLE: 'tmd-last-table',
  PREFERENCES: 'tmd-preferences'
} as const

// Error Messages
export const ERROR_MESSAGES = {
  NETWORK_ERROR: '网络连接失败，请检查网络设置',
  SERVER_ERROR: '服务器内部错误，请稍后重试',
  UNAUTHORIZED: '未授权，请重新登录',
  FORBIDDEN: '没有权限执行此操作',
  NOT_FOUND: '请求的资源不存在',
  TOO_MANY_REQUESTS: '请求过于频繁，请稍后再试',
  VALIDATION_ERROR: '数据验证失败',
  UNKNOWN_ERROR: '未知错误，请重试'
} as const

// Success Messages
export const SUCCESS_MESSAGES = {
  CREATED: '创建成功',
  UPDATED: '更新成功',
  DELETED: '删除成功',
  SAVED: '保存成功',
  COPIED: '已复制到剪贴板',
  DOWNLOADED: '下载完成'
} as const

// Log Levels
export const LOG_LEVELS = {
  DEBUG: 'DEBUG' as const,
  INFO: 'INFO' as const,
  WARNING: 'WARNING' as const,
  ERROR: 'ERROR' as const
} as const

export const LOG_LEVEL_COLORS: Record<string, string> = {
  [LOG_LEVELS.DEBUG]: '#6e7681',
  [LOG_LEVELS.INFO]: '#58a6ff',
  [LOG_LEVELS.WARNING]: '#f0883e',
  [LOG_LEVELS.ERROR]: '#f85149'
}
