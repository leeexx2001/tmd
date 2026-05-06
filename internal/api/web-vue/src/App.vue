<template>
  <div id="app-wrapper">
    <router-view />
    <BaseToast :toasts="toast.toasts.value" @remove="toast.removeToast" />
  </div>
</template>

<script setup lang="ts">
import { toast } from '@/composables/useToast'
import BaseToast from '@/components/common/BaseToast.vue'
</script>

<style>
/* Global styles are imported in main.ts */
</style>
