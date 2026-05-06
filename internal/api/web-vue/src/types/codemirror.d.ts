// CodeMirror type declarations for missing modules
declare module 'codemirror/mode/yaml/yaml' {
  import { CodeMirror } from 'codemirror'
  const yaml: CodeMirror.Mode<any>
  export default yaml
}

declare module 'codemirror/theme/material-darker.css' {
  const css: string
  export default css
}
