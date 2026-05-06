<template>
  <div class="mobile-card-list">
    <div
      v-for="item in items"
      :key="getRowKey(item)"
      class="mobile-card"
    >
      <div class="card-header">
        <h3 class="card-title">{{ getCardTitle(item) }}</h3>
        <span :class="['status-badge', getStatusClass(item)]">
          {{ getStatusText(item) }}
        </span>
      </div>
      
      <div class="card-fields">
        <div
          v-for="field in visibleFields"
          :key="field.key"
          class="field-row"
        >
          <span class="field-label">{{ field.label }}:</span>
          <span class="field-value">{{ formatFieldValue(item, field.key) }}</span>
        </div>
      </div>

      <div class="card-actions" v-if="showActions">
        <button
          v-if="editable"
          class="btn btn-primary btn-sm"
          @click="$emit('edit', item)"
        >
          ✏️ 编辑
        </button>
        <button
          v-if="deletable"
          class="btn btn-danger btn-sm"
          @click="$emit('delete', item)"
        >
          🗑️ 删除
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { User, List, UserEntity, UserLink } from '@/types'

const props = withDefaults(defineProps<{
  items: any[]
  columns: { key: string; label: string }[]
  tableType: 'users' | 'lists' | 'entities' | 'listEntities' | 'userLinks'
  editable?: boolean
  deletable?: boolean
  showActions?: boolean
}>(), {
  editable: true,
  deletable: true,
  showActions: true
})

const emit = defineEmits<{
  edit: [item: any]
  delete: [item: any]
}>()

// Show only important fields on mobile (exclude ID and long text fields)
const visibleFields = computed(() => {
  const excludeFields = ['id', 'description', 'profile_image_url', 'thumbnail_url', 'file_path']
  return props.columns.filter(col => !excludeFields.includes(col.key))
})

function getRowKey(item: any): number {
  return item.id || item.user_id || Math.random()
}

function getCardTitle(item: any): string {
  switch (props.tableType) {
    case 'users':
      return (item as User).username || (item as User).display_name || `User #${item.id}`
    case 'lists':
      return (item as List).name || `List #${item.id}`
    case 'entities':
      return `${(item as UserEntity).entity_type} #${item.id}`
    case 'listEntities':
      return `List Entity #${item.id}`
    case 'userLinks':
      return `${(item as UserLink).link_type} #${item.id}`
    default:
      return `Item #${item.id}`
  }
}

function formatFieldValue(item: any, key: string): string {
  const value = item[key]
  
  if (value === null || value === undefined) return '-'
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (typeof value === 'number') {
    // Format large numbers
    if (value > 1000000) return `${(value / 1000000).toFixed(1)}M`
    if (value > 1000) return `${(value / 1000).toFixed(1)}K`
    return value.toLocaleString()
  }
  if (typeof value === 'string') {
    // Truncate long strings
    if (value.length > 50) return `${value.substring(0, 50)}...`
    return value
  }
  
  return String(value)
}

function getStatusClass(item: any): string {
  // Add status-based styling based on item properties
  if ((item as User).is_verified) return 'verified'
  if ((item as User).is_protected) return 'protected'
  if ((item as List).is_private) return 'private'
  return ''
}

function getStatusText(item: any): string {
  if ((item as User).is_verified) return '✓ 已认证'
  if ((item as User).is_protected) return '🔒 受保护'
  if ((item as List).is_private) return '🔒 私有'
  return ''
}
</script>

<style scoped lang="scss">
.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-2) 0;
}

.mobile-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  transition: all var(--duration-fast);

  &:active {
    border-color: var(--accent-primary);
    transform: scale(0.98);
  }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    margin-bottom: var(--space-3);
    padding-bottom: var(--space-3);
    border-bottom: 1px solid var(--border-primary);
  }

  .card-title {
    font-size: var(--text-base);
    font-weight: var(--font-semibold);
    color: var(--text-primary);
    margin: 0;
    flex: 1;
    min-width: 0;
    
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .status-badge {
    display: inline-flex;
    align-items: center;
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-full);
    font-size: var(--text-xs);
    font-weight: var(--font-medium);
    white-space: nowrap;

    &.verified {
      background: var(--success-bg);
      color: var(--success);
    }

    &.protected,
    &.private {
      background: var(--warning-bg);
      color: var(--warning);
    }
  }

  .card-fields {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    margin-bottom: var(--space-4);
  }

  .field-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2) 0;
  }

  .field-label {
    font-size: var(--text-sm);
    color: var(--text-tertiary);
    min-width: 80px;
    flex-shrink: 0;
  }

  .field-value {
    font-size: var(--text-sm);
    color: var(--text-primary);
    text-align: right;
    word-break: break-all;
    flex: 1;
  }

  .card-actions {
    display: flex;
    gap: var(--space-2);
    justify-content: flex-end;
    padding-top: var(--space-3);
    border-top: 1px solid var(--border-primary);

    .btn {
      flex: 1;
      max-width: 120px;
    }
  }
}
</style>
