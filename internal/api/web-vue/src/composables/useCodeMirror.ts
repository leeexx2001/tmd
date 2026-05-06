// useCodeMirror - CodeMirror Editor Composable for YAML Editing
import { ref, onMounted, onUnmounted, watch, type Ref } from 'vue'

export function useCodeMirror(options: {
  content: Ref<string>
  mode?: string
  theme?: string
  readOnly?: boolean
  lineNumbers?: boolean
}) {
  const containerRef = ref<HTMLElement | null>(null)
  let editor: any = null

  const defaultOptions = {
    mode: options.mode || 'yaml',
    theme: options.theme || 'material-darker',
    lineNumbers: options.lineNumbers !== false,
    readOnly: options.readOnly || false,
    tabSize: 2,
    indentWithTabs: false,
    lineWrapping: true,
    autofocus: false,
    extraKeys: {
      'Tab': (cm: any) => {
        if (cm.somethingSelected()) {
          cm.indentSelection('add')
        } else {
          cm.replaceSelection('  ', 'end')
        }
      }
    }
  }

  onMounted(async () => {
    if (!containerRef.value) return

    // Dynamic import CodeMirror to avoid SSR issues
    const CodeMirror = (await import('codemirror')).default
    
    // Import YAML mode
    await import('codemirror/mode/yaml/yaml')
    
    // Import theme
    await import('codemirror/theme/material-darker.css')

    editor = CodeMirror(containerRef.value, {
      ...defaultOptions,
      value: options.content.value
    })

    // Sync editor changes to reactive variable
    editor.on('change', () => {
      if (options.content.value !== editor.getValue()) {
        options.content.value = editor.getValue()
      }
    })

    // Auto-resize on window resize
    const resizeObserver = new ResizeObserver(() => {
      if (editor) {
        editor.refresh()
      }
    })
    
    resizeObserver.observe(containerRef.value)
  })

  // Watch external content changes and sync to editor
  watch(() => options.content.value, (newVal) => {
    if (editor && editor.getValue() !== newVal) {
      editor.setValue(newVal)
    }
  })

  onUnmounted(() => {
    if (editor) {
      editor.toTextArea()
      editor = null
    }
  })

  return {
    containerRef,
    getEditor: () => editor
  }
}
