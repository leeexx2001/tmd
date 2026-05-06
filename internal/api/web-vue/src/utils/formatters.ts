// Formatters - Data Formatting Utilities

/**
 * Format file size to human-readable string
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return `${(bytes / Math.pow(k, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

/**
 * Format number with locale-specific formatting
 */
export function formatNumber(num: number): string {
  if (num >= 1000000) {
    return `${(num / 1000000).toFixed(1)}M`
  } else if (num >= 1000) {
    return `${(num / 1000).toFixed(1)}K`
  }
  return num.toLocaleString('zh-CN')
}

/**
 * Format date/time to relative or absolute string
 */
export function formatTime(dateStr?: string | Date): string {
  if (!dateStr) return '-'
  
  const date = typeof dateStr === 'string' ? new Date(dateStr) : dateStr
  
  if (isNaN(date.getTime())) return '-'
  
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)
  
  // Relative time for recent dates
  if (diffMins < 1) return '刚刚'
  if (diffMins < 60) return `${diffMins}分钟前`
  if (diffHours < 24) return `${diffHours}小时前`
  if (diffDays < 7) return `${diffDays}天前`
  
  // Absolute time for older dates
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

/**
 * Format duration in milliseconds to human-readable string
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`
  
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}h ${remainingMinutes}m`
}

/**
 * Format percentage value
 */
export function formatPercent(value: number, decimals: number = 1): string {
  return `${value.toFixed(decimals)}%`
}

/**
 * Truncate text with ellipsis
 */
export function truncate(text: string, maxLength: number): string {
  if (!text || text.length <= maxLength) return text
  return text.substring(0, maxLength) + '...'
}

/**
 * Escape HTML special characters
 */
export function escapeHtml(text: string): string {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

/**
 * Capitalize first letter
 */
export function capitalize(text: string): string {
  if (!text) return ''
  return text.charAt(0).toUpperCase() + text.slice(1)
}

/**
 * Convert snake_case to Title Case
 */
export function snakeToTitle(snake: string): string {
  return snake
    .split('_')
    .map(word => capitalize(word))
    .join(' ')
}

/**
 * Format cron expression to human-readable description
 */
export function formatCronExpression(cron: string): string {
  try {
    const parts = cron.split(/\s+/)
    if (parts.length !== 5) return cron
    
    const [minute, hour, dayOfMonth, month, dayOfWeek] = parts
    
    // Simple patterns
    if (minute === '0' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
      return '每小时'
    }
    
    if (minute === '0' && hour !== '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
      return `每天 ${hour.padStart(2, '0')}:00`
    }
    
    if (minute.includes('/') && hour === '*') {
      const interval = minute.split('/')[1]
      return `每 ${interval} 分钟`
    }
    
    return cron
  } catch {
    return cron
  }
}
