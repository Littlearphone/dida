/**
 * 章节拆分逻辑 — 从 NovelEditor 提取
 */
import { ref, computed } from 'vue'
import { DOMSerializer } from '@tiptap/pm/model'
import { useNovelStore } from '../stores/novel'
import { useMessage } from 'naive-ui'

export function useChapterSplit(editor: any, doSaveChapter: () => Promise<boolean>) {
  const novelStore = useNovelStore()
  const message = useMessage()

  const showSplitDialog = ref(false)
  const splitChapterTitle = ref('')
  const splittingChapter = ref(false)

  /** 编辑器是否包含非空选区 */
  const hasSelection = computed(() => {
    const ed = editor.value
    if (!ed) return false
    const { from, to } = ed.state.selection
    return from !== to
  })

  function getSelectedHtml(): string {
    const ed = editor.value
    if (!ed) return ''
    const { state } = ed
    const { from, to } = state.selection
    if (from === to) return ''
    const slice = state.doc.slice(from, to)
    const serializer = DOMSerializer.fromSchema(state.schema)
    const fragment = serializer.serializeFragment(slice.content)
    const tempDiv = document.createElement('div')
    tempDiv.appendChild(fragment)
    return tempDiv.innerHTML
  }

  function handleSplitClick() {
    const ed = editor.value
    const ch = novelStore.currentChapter
    if (!ed || !ch) return

    const selectedHtml = getSelectedHtml()
    if (!selectedHtml) {
      message.warning('请先在正文中选择要拆分的文本')
      return
    }

    splitChapterTitle.value = `${ch.title || '新章'}（拆出）`
    showSplitDialog.value = true
  }

  async function confirmSplit(title: string) {
    if (!title) { message.warning('请输入章节标题'); return }

    const ed = editor.value
    const ch = novelStore.currentChapter
    const n = novelStore.currentNovel
    if (!ed || !ch || !n) return

    const selectedHtml = getSelectedHtml()
    if (!selectedHtml) { message.warning('请先选择要拆分的文本'); return }

    splittingChapter.value = true

    // 1. 从当前章节删除选中内容
    const { state, view } = ed
    const tr = state.tr.deleteSelection()
    view.dispatch(tr)
    view.focus()
    await doSaveChapter()

    // 2. 创建新章节
    const newCh = await novelStore.createChapter({
      novelId: n.id,
      title,
      content: selectedHtml,
      order: novelStore.chapters.length + 1,
    })
    if (!newCh) { message.error('创建章节失败'); splittingChapter.value = false; return }

    // 3. 重新排序：放到当前章节之后
    const currentIdx = novelStore.chapters.findIndex(c => c.id === ch.id)
    const ids = novelStore.chapters.filter(c => c.id !== newCh.id).map(c => c.id)
    ids.splice(currentIdx + 1, 0, newCh.id)
    await novelStore.reorderChapters(n.id, ids)

    // 4. 跳转到新章节
    await novelStore.loadChapters(n.id)
    const found = novelStore.chapters.find(c => c.id === newCh.id)
    if (found) novelStore.selectChapter(found)
    showSplitDialog.value = false
    splittingChapter.value = false
    message.success('已拆分为新章节')
  }

  function cancelSplit() {
    splitChapterTitle.value = ''
  }

  return {
    showSplitDialog, splitChapterTitle, splittingChapter,
    hasSelection,
    handleSplitClick, confirmSplit, cancelSplit,
  }
}
