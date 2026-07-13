/**
 * 章节列表拖拽排序 — 从 ChapterSidebar 提取
 */
import { ref } from 'vue'
import { useNovelStore } from '../stores/novel'

export function useChapterDrag() {
  const novelStore = useNovelStore()
  const dragIndex = ref<number | null>(null)
  const dragOverIndex = ref<number | null>(null)

  function handleDragStart(i: number) { dragIndex.value = i }

  function handleDragOver(e: DragEvent, i: number) {
    e.preventDefault()
    const list = novelStore.chapters
    if (i === list.length - 1) {
      const target = e.currentTarget as HTMLElement
      if (target) {
        const rect = target.getBoundingClientRect()
        if (e.clientY - rect.top > rect.height * 0.55) {
          dragOverIndex.value = list.length; return
        }
      }
    }
    dragOverIndex.value = i
  }

  function handleDragLeave() { dragOverIndex.value = null }

  async function handleDrop(e: DragEvent) {
    e.preventDefault()
    if (dragIndex.value === null || dragIndex.value === dragOverIndex.value) {
      dragIndex.value = null; dragOverIndex.value = null; return
    }
    const list = [...novelStore.chapters]
    const from = dragIndex.value
    const to = dragOverIndex.value ?? novelStore.chapters.length - 1
    if (from === to) { dragIndex.value = null; dragOverIndex.value = null; return }
    const [moved] = list.splice(from, 1)
    const adjustedTo = to > from ? to - 1 : to
    list.splice(adjustedTo, 0, moved)
    const nid = novelStore.currentNovel?.id
    if (!nid) return
    if (await novelStore.reorderChapters(nid, list.map(ch => ch.id))) {
      await novelStore.loadChapters(nid)
    }
    dragIndex.value = null; dragOverIndex.value = null
  }

  function handleDragEnd() { dragIndex.value = null; dragOverIndex.value = null }

  return {
    dragIndex, dragOverIndex,
    handleDragStart, handleDragOver, handleDragLeave,
    handleDrop,
    handleDragEnd,
  }
}
