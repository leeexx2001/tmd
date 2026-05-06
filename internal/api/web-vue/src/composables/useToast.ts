// useToast - Toast Notification System Composable
import { ref } from 'vue'

export interface ToastMessage {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  message: string
  duration?: number
  action?: {
    label: string
    onClick: () => void
  }
}

let toastId = 0

export function useToast() {
  const toasts = ref<ToastMessage[]>([])

  function addToast(toast: Omit<ToastMessage, 'id'>) {
    const id = `toast-${++toastId}`
    const newToast: ToastMessage = {
      ...toast,
      id,
      duration: toast.duration ?? (toast.type === 'error' ? 0 : 4000)
    }

    toasts.value = [...toasts.value, newToast]

    if (newToast.duration && newToast.duration > 0) {
      setTimeout(() => {
        removeToast(id)
      }, newToast.duration)
    }

    return id
  }

  function removeToast(id: string) {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }

  function success(message: string, options?: Partial<Omit<ToastMessage, 'id' | 'type' | 'message'>>) {
    return addToast({ type: 'success', message, ...options })
  }

  function error(message: string, options?: Partial<Omit<ToastMessage, 'id' | 'type' | 'message'>>) {
    return addToast({ type: 'error', message, ...options })
  }

  function warning(message: string, options?: Partial<Omit<ToastMessage, 'id' | 'type' | 'message'>>) {
    return addToast({ type: 'warning', message, ...options })
  }

  function info(message: string, options?: Partial<Omit<ToastMessage, 'id' | 'type' | 'message'>>) {
    return addToast({ type: 'info', message, ...options })
  }

  function clearAll() {
    toasts.value = []
  }

  return {
    toasts,
    addToast,
    removeToast,
    success,
    error,
    warning,
    info,
    clearAll
  }
}

// Global toast instance for convenience
const globalToast = useToast()

export const toast = globalToast
