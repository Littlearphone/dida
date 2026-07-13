<script setup lang="ts">
/**
 * 编辑器内容区 — 包裹 Tiptap 编辑器的纸面容器 + crop marks + 拆分章节按钮
 */
import { EditorContent } from '@tiptap/vue-3'
import { NButton, NIcon, NText } from 'naive-ui'
import { CutOutline as SplitIcon } from '@vicons/ionicons5'

defineProps<{
  currentChapter: any
  editor: any
  editorStyles: Record<string, any>
  hasSelection: boolean
}>()

const emit = defineEmits<{
  split: []
}>()
</script>

<template>
  <div class="editor-area">
    <div v-if="currentChapter" class="editor-content" :style="editorStyles">
      <div class="editor-page">
        <editor-content :editor="editor" class="content-editable" />
        <!-- 拆分章节按钮栏 -->
        <div class="split-bar">
          <n-button size="tiny" round
            :type="hasSelection ? 'primary' : 'default'"
            :disabled="!hasSelection"
            @click="emit('split')"
            title="将选中内容拆分为新章节">
            <template #icon><n-icon size="16"><SplitIcon /></n-icon></template>
            拆分为新章节
          </n-button>
        </div>
      </div>
    </div>
    <div v-else class="editor-empty">
      <n-text depth="3">还没有章节，请创建第一章</n-text>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.editor-area {
  flex: 1; overflow: hidden; min-height: 0; display: flex;
}

.editor-content {
  flex: 1; overflow-y: auto; background: #f0f2f5;
}

/* Word 风格的"页面"容器：白底、阴影、四角对齐标记 */
.editor-page {
  width: 100%; max-width: 960px; min-height: 100%;
  margin: 0 auto; background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  position: relative;
  display: flex; flex-direction: column;

  /* ── 四角对齐标记（Word 风格 crop marks）── */
  &::before {
    content: ''; position: absolute; pointer-events: none; z-index: 1;
    top: 28px; left: 44px;
    width: 20px; height: 20px;
    border-right: 2px solid #c0c0c0;
    border-bottom: 2px solid #c0c0c0;
  }

  &::after {
    content: ''; position: absolute; pointer-events: none; z-index: 1;
    top: 28px; right: 44px;
    width: 20px; height: 20px;
    border-left: 2px solid #c0c0c0;
    border-bottom: 2px solid #c0c0c0;
  }
}

.content-editable {
  padding: 48px 64px 64px;
  outline: none;
  white-space: pre-wrap; box-sizing: border-box;
  position: relative;
  flex: 1;
  display: flex; flex-direction: column;

  &:focus,
  &:focus-visible,
  &:focus-within {
    outline: none;
  }

  /* ── 下角 crop marks ── */
  &::before {
    content: ''; position: absolute; pointer-events: none; z-index: 1;
    bottom: 44px; left: 44px;
    width: 20px; height: 20px;
    border-right: 2px solid #c0c0c0;
    border-top: 2px solid #c0c0c0;
  }

  &::after {
    content: ''; position: absolute; pointer-events: none; z-index: 1;
    bottom: 44px; right: 44px;
    width: 20px; height: 20px;
    border-left: 2px solid #c0c0c0;
    border-top: 2px solid #c0c0c0;
  }

  :deep(p) {
    text-indent: 2em;
    margin-bottom: var(--p-gap, 16px);
  }

  :deep(p:last-child) {
    margin-bottom: 0;
  }

  /* 覆盖 ProseMirror 默认 focus 样式 */
  :deep(.ProseMirror),
  :deep(.ProseMirror-focused),
  :deep(.ProseMirror:focus) {
    outline: none !important;
    border: none !important;
    box-shadow: none !important;
    flex: 1;
  }
}

/* 拆分章节按钮栏 */
.split-bar {
  flex-shrink: 0;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 8px 64px 12px;
  gap: 8px;
}

.editor-empty {
  flex: 1; display: flex; align-items: center; justify-content: center; background: #f0f2f5;
}
</style>
