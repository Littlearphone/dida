import type { InjectionKey } from 'vue'

/** AI 对话弹框通过此接口操作编辑器 */
export interface EditorActions {
  setContent: (html: string) => void
  getContent: () => string
  markChanged: () => void
  /** 获取当前选区文本（通过 ProseMirror 选区，不受浏览器焦点影响） */
  getSelectionText?: () => string
  /** 将当前选区替换为指定文本 */
  replaceSelection?: (text: string) => void
}

export const EDITOR_ACTIONS_KEY: InjectionKey<EditorActions> = Symbol('editorActions')
