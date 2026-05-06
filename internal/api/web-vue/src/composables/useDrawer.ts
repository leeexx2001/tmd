// useDrawer - Unified Drawer Control Composable
import { ref, provide, inject } from 'vue'

const DRAWER_KEY = Symbol('drawer')

export interface DrawerState {
  visible: boolean
  title: string
  footer: boolean
  component?: any
  props?: Record<string, any>
}

export function useDrawerProvider() {
  const drawer = ref<DrawerState>({
    visible: false,
    title: '',
    footer: false
  })

  function open(options: Partial<DrawerState> & { title: string }) {
    drawer.value = {
      visible: true,
      title: options.title,
      footer: options.footer ?? false,
      component: options.component,
      props: options.props
    }
  }

  function close() {
    drawer.value.visible = false
    // Reset after animation
    setTimeout(() => {
      drawer.value.component = undefined
      drawer.value.props = undefined
    }, 300)
  }

  function toggle() {
    if (drawer.value.visible) {
      close()
    } else {
      drawer.value.visible = true
    }
  }

  provide(DRAWER_KEY, {
    drawer,
    open,
    close,
    toggle
  })

  return {
    drawer,
    open,
    close,
    toggle
  }
}

export function useDrawer() {
  const injected = inject<ReturnType<typeof useDrawerProvider>>(DRAWER_KEY)
  
  if (!injected) {
    console.warn('[useDrawer] Drawer context not found. Make sure useDrawerProvider() is called in a parent component.')
    
    // Fallback to local state
    const drawer = ref<DrawerState>({ visible: false, title: '', footer: false })
    
    return {
      drawer,
      open: (options: Partial<DrawerState> & { title: string }) => {
        Object.assign(drawer.value, options, { visible: true })
      },
      close: () => { drawer.value.visible = false },
      toggle: () => { drawer.value.visible = !drawer.value.visible }
    }
  }
  
  return injected
}
