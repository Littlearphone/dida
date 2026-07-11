import type { InjectionKey } from 'vue'

/** AI 对话弹框通过此接口操作编辑器 */
export interface EditorActions {
  setContent: (html: string) => void
  getContent: () => string
  markChanged: () => void
}

export const EDITOR_ACTIONS_KEY: InjectionKey<EditorActions> = Symbol('editorActions')
