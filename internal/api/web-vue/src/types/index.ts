// TMD Vue Frontend - Type Definitions

// ===================================
// API Response Types
// ===================================
export interface ApiResponse<T = any> {
  success: boolean
  data: T
  error?: string
  message?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

// ===================================
// Task Types
// ===================================
export interface TaskProgress {
  total?: number
  completed?: number
  stage?: string
}

export interface TaskData {
  screen_name?: string
  list_id?: string | number
  users?: string[]
  lists?: (string | number)[]
  [key: string]: any
}

export interface Task {
  task_id: string
  type: 'list' | 'user' | 'following' | 'likes' | 'media'
  target: string
  status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'
  progress: TaskProgress
  data?: TaskData
  total_items?: number
  processed_items?: number
  failed_items?: number
  created_at: string
  started_at?: string
  completed_at?: string
  error_message?: string
  options?: TaskOptions
}

export interface TaskOptions {
  download_media?: boolean
  media_quality?: 'high' | 'medium' | 'low'
  skip_profile?: boolean
  auto_follow?: boolean
  no_retry?: boolean
  max_concurrent?: number
}

export interface Health {
  status: 'healthy' | 'degraded' | 'unhealthy'
  version: string
  uptime: number
  active_tasks: number
  total_tasks_completed: number
  database_connected: boolean
  scheduler_running: boolean
}

// ===================================
// Database Entity Types
// ===================================
export interface User {
  id: number
  username: string
  display_name?: string
  description?: string
  followers_count?: number
  following_count?: number
  tweets_count?: number
  likes_count?: number
  profile_image_url?: string
  created_at?: string
  updated_at?: string
  last_synced_at?: string
  is_verified?: boolean
  is_protected?: boolean
}

export interface List {
  id: number
  list_id: string
  name: string
  description?: string
  owner_id?: number
  owner_username?: string
  member_count?: number
  subscriber_count?: number
  created_at?: string
  updated_at?: string
  is_private?: boolean
}

export interface UserEntity {
  id: number
  user_id: number
  entity_type: 'tweet' | 'media' | 'video' | 'gif'
  entity_id: string
  url?: string
  thumbnail_url?: string
  media_type?: string
  file_path?: string
  file_size?: number
  width?: number
  height?: number
  duration_ms?: number
  alt_text?: string
  downloaded_at?: string
  created_at?: string
}

export interface ListEntity {
  id: number
  list_id: number
  user_entity_id: number
  added_at?: string
}

export interface UserLink {
  id: number
  source_user_id: number
  target_user_id: number
  link_type: 'following' | 'follower' | 'blocking' | 'muting'
  created_at?: string
}

export interface PaginationData<T = any> {
  data: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

// ===================================
// Configuration Types
// ===================================
export interface ConfigField {
  name: string
  label: string
  type: 'string' | 'number' | 'boolean' | 'select'
  value: string | number | boolean
  default_value?: string | number | boolean
  placeholder?: string
  prompt?: string
  options?: { label: string; value: string }[]
  required?: boolean
  group?: string
}

export interface CookieItem {
  auth_token: string
  ct0: string
  added_at?: string
  last_used_at?: string
  user_id?: string
  screen_name?: string
}

// ===================================
// Schedule Types
// ===================================
export interface ScheduleEntry {
  id: string
  name?: string
  type: 'list' | 'user' | 'following'
  target: string
  schedule: string
  enabled: boolean
  run_on_start?: boolean
  auto_follow?: boolean
  skip_profile?: boolean
  no_retry?: boolean
  max_retries?: number
  retry_delay?: number
  options?: TaskOptions
  created_at?: string
  updated_at?: string
  last_run_by?: string
}

export interface ScheduleStatus {
  entry: ScheduleEntry
  schedule_display: string
  next_run_at?: string
  last_run_at?: string
  run_count: number
  consecutive_failures: number
  last_error?: string
  average_duration_ms?: number
  is_running: boolean
}

// ===================================
// Log Types
// ===================================
export interface LogEntry {
  timestamp: string
  level: 'DEBUG' | 'INFO' | 'WARNING' | 'ERROR'
  message: string
  module?: string
  context?: Record<string, any>
}

// ===================================
// UI State Types
// ===================================
export interface SidebarState {
  collapsed: boolean
  activeItem: string
}

export interface AppSettings {
  theme: 'light' | 'dark' | 'system'
  language: 'zh-CN' | 'en-US'
  density: 'comfortable' | 'compact' | 'spacious'
  notifications: boolean
  soundEffects: boolean
}

export interface FilterState {
  status: Task['status'] | 'all'
  type: Task['type'] | 'all'
  search: string
  dateRange?: [string, string]
}

export interface SortState {
  sortBy: string
  sortOrder: 'asc' | 'desc'
}
