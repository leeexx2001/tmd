<template>
  <div class="not-found-page">
    <div class="not-found-content">
      <div class="error-code">404</div>
      <h1 class="error-title">页面未找到</h1>
      <p class="error-description">
        抱歉，您访问的页面不存在或已被移除。
      </p>
      <div class="error-actions">
        <router-link to="/" class="btn btn-primary">
          🏠 返回首页
        </router-link>
        <button class="btn btn-secondary" @click="$router.back()">
          ← 返回上页
        </button>
      </div>
      
      <div class="help-section">
        <h3>可能的原因：</h3>
        <ul>
          <li>URL地址输入错误</li>
          <li>该页面已被删除或移动</li>
          <li>链接已过期</li>
        </ul>
        
        <h3 style="margin-top: var(--space-4);">快速导航：</h3>
        <nav class="quick-links">
          <router-link to="/" class="quick-link">📊 概览</router-link>
          <router-link to="/tasks" class="quick-link">🚀 任务</router-link>
          <router-link to="/data/users" class="quick-link">👥 数据</router-link>
          <router-link to="/schedules" class="quick-link">⏰ 定时任务</router-link>
          <router-link to="/system/config" class="quick-link">⚙️ 系统</router-link>
        </nav>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'

onMounted(() => {
  console.warn('[Router] 页面未找到:', window.location.pathname)
})
</script>

<style scoped lang="scss">
.not-found-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: calc(100vh - var(--header-height));
  padding: var(--space-8);
}

.not-found-content {
  max-width: 600px;
  text-align: center;
}

.error-code {
  font-size: 120px;
  font-weight: var(--font-bold);
  line-height: 1;
  background: linear-gradient(135deg, var(--color-primary), var(--color-warning));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: var(--space-4);
  opacity: 0.8;
}

.error-title {
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
  margin-bottom: var(--space-3);
}

.error-description {
  font-size: var(--text-base);
  color: var(--text-secondary);
  line-height: 1.6;
  margin-bottom: var(--space-8);
}

.error-actions {
  display: flex;
  gap: var(--space-3);
  justify-content: center;
  margin-bottom: var(--space-12);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-6);
  border-radius: var(--radius-md);
  font-size: var(--text-base);
  font-weight: var(--font-medium);
  transition: all var(--duration-fast);
  text-decoration: none;

  &-primary {
    background: var(--color-primary);
    color: white;

    &:hover {
      background: var(--color-primary-hover);
      text-decoration: none;
    }
  }

  &-secondary {
    background: var(--bg-tertiary);
    color: var(--text-secondary);

    &:hover {
      background: var(--border-primary);
      color: var(--text-primary);
    }
  }
}

.help-section {
  text-align: left;
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--space-6);

  h3 {
    font-size: var(--text-base);
    font-weight: var(--font-semibold);
    color: var(--text-primary);
    margin-bottom: var(--space-3);
  }

  ul {
    list-style: disc;
    padding-left: var(--space-6);
    color: var(--text-secondary);
    font-size: var(--text-sm);
    line-height: 1.8;

    li {
      margin-bottom: var(--space-1);
    }
  }
}

.quick-links {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-top: var(--space-3);
}

.quick-link {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-2) var(--space-3);
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--text-secondary);
  text-decoration: none;
  transition: all var(--duration-fast);

  &:hover {
    background: var(--color-primary-light);
    border-color: var(--color-primary);
    color: var(--color-primary);
    text-decoration: none;
  }
}

@media (max-width: 767px) {
  .error-code {
    font-size: 80px;
  }

  .error-title {
    font-size: var(--text-2xl);
  }

  .error-actions {
    flex-direction: column-reverse;

    .btn {
      width: 100%;
    }
  }

  .quick-links {
    flex-direction: column;

    .quick-link {
      justify-content: center;
    }
  }
}
</style>
