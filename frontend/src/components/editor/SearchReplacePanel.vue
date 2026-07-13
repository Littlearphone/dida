<script setup lang="ts">
/**
 * 搜索/替换面板 — UI 层，核心逻辑由 useSearch 管理
 */
import { onMounted, onUnmounted } from 'vue'
import { useSearch } from '../../composables/useSearch'
import { NButton, NIcon, NInput, NText } from 'naive-ui'
import {
  ChevronUpOutline as PrevIcon,
  ChevronDownOutline as NextIcon,
  CloseOutline as CloseIcon,
} from '@vicons/ionicons5'

const props = defineProps<{
  editor: any
  doSaveChapter: () => Promise<boolean>
}>()

// 解构 composable 为顶层绑定，模板中自动解包 ref
const s = useSearch(() => props.editor, props.doSaveChapter)

const {
  showSearch, showReplace, searchQuery, replaceText,
  currentMatchIndex, totalMatches, searchAll,
  allChapterMatches, allSearchTotal,
  updateSearch, findNext, findPrev, closeSearch,
  replaceCurrent, replaceAll, replaceAllInBook,
  openSearch, openReplace, fillSearchFromSelection,
  navigateToChapterSearch,
  registerPlugin, unregisterPlugin,
} = s

function handleKeydown(e: KeyboardEvent) {
  if (!s.isOpen()) return
  const isCtrl = e.ctrlKey || e.metaKey
  if (e.key === 'F3' || (isCtrl && e.key === 'g') ||
      (e.key === 'Enter' && (e.target as HTMLElement).closest('.search-bar'))) {
    e.preventDefault()
    if (e.shiftKey) findPrev(); else findNext()
    return
  }
  if (e.key === 'Escape') { closeSearch(); e.preventDefault(); return }
}

defineExpose({
  isOpen: s.isOpen,
  isReplaceOpen: s.isReplaceOpen,
  openSearch, openReplace, closeSearch,
  findNext, findPrev,
  fillSearchFromSelection,
})

onMounted(() => { registerPlugin(); document.addEventListener('keydown', handleKeydown) })
onUnmounted(() => { unregisterPlugin(); document.removeEventListener('keydown', handleKeydown) })
</script>

<template>
  <div v-if="showSearch" class="search-bar">
    <div class="search-bar-inner">
      <div class="search-row">
        <div class="search-scope">
          <span class="scope-btn" :class="{ active: !searchAll }"
            @click="searchAll = false; updateSearch()">本章</span>
          <span class="scope-divider">|</span>
          <span class="scope-btn" :class="{ active: searchAll }"
            @click="searchAll = true; updateSearch()">全书</span>
        </div>
        <n-input v-model:value="searchQuery" placeholder="搜索正文..." size="small"
          class="search-input" style="width:200px"
          @update:value="updateSearch()"
          @keydown.enter="findNext()" />

        <n-button quaternary size="tiny" :disabled="totalMatches === 0"
          @click="findPrev()" title="上一个 (Shift+F3)">
          <template #icon><n-icon size="14"><PrevIcon/></n-icon></template>
        </n-button>
        <n-button quaternary size="tiny" :disabled="totalMatches === 0"
          @click="findNext()" title="下一个 (F3)">
          <template #icon><n-icon size="14"><NextIcon/></n-icon></template>
        </n-button>

        <n-text v-if="totalMatches > 0" class="match-counter">
          {{ currentMatchIndex }}/{{ totalMatches }}
        </n-text>
        <template v-if="searchAll && allSearchTotal > 0">
          <n-text depth="3" class="match-counter" style="margin-left:0">（全 {{ allSearchTotal }}）</n-text>
        </template>
        <n-text v-if="totalMatches === 0 && !(searchAll && allSearchTotal > 0)"
          depth="3" class="match-counter">无结果</n-text>

        <n-button quaternary size="tiny" class="search-close" @click="closeSearch()" title="关闭搜索 (Esc)">
          <template #icon><n-icon size="14"><CloseIcon/></n-icon></template>
        </n-button>
      </div>

      <div v-if="showReplace" class="search-row">
        <span class="search-scope-placeholder" aria-hidden="true">
          <span class="scope-btn">本章</span>
          <span class="scope-divider">|</span>
          <span class="scope-btn">全书</span>
        </span>
        <n-input v-model:value="replaceText" placeholder="替换为..." size="small"
          class="replace-input" style="width:200px" />

        <template v-if="!searchAll">
          <n-button size="tiny" :disabled="totalMatches === 0 || !replaceText"
            @click="replaceCurrent()">替换</n-button>
          <n-button size="tiny" :disabled="totalMatches === 0 || !replaceText"
            @click="replaceAll()">全部替换</n-button>
        </template>
        <template v-else>
          <n-button size="tiny" :disabled="allSearchTotal === 0 || !replaceText"
            @click="replaceAllInBook()">全书替换</n-button>
        </template>
      </div>

      <div v-if="searchAll && allChapterMatches.length > 0" class="all-search-results">
        <div v-for="cm in allChapterMatches" :key="cm.chapterId" class="all-search-chapter">
          <div class="all-search-chapter-title" @click="navigateToChapterSearch(cm.chapterId)">
            {{ cm.chapterTitle }}（{{ cm.total }} 处）
          </div>
          <div v-for="(s, si) in cm.snippets.slice(0, 5)" :key="si"
            class="all-search-snippet" @click="navigateToChapterSearch(cm.chapterId)">
            <span class="snippet-before">{{ s.before }}</span>
            <span class="snippet-match">{{ s.match }}</span>
            <span class="snippet-after">{{ s.after }}</span>
          </div>
          <div v-if="cm.snippets.length > 5" class="all-search-more">还有 {{ cm.snippets.length - 5 }} 处…</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.search-bar {
  background: #fafafa;
  border-bottom: 1px solid #eee;
  padding: 6px 16px;
  flex-shrink: 0;
  display: flex; justify-content: center;
}
.search-bar-inner {
  max-width: 960px;
  width: 100%;
  padding: 0 64px;
  display: flex; flex-direction: column; gap: 6px;
}
.search-row {
  display: flex; align-items: center; gap: 6px;
  flex-wrap: wrap;
}
.match-counter {
  font-size: 12px; min-width: 44px;
  text-align: center; white-space: nowrap;
}
.search-close { margin-left: 4px; }
.search-scope {
  display: flex; align-items: center; gap: 4px;
  font-size: 12px; color: #888; user-select: none;
  flex-shrink: 0;
}
.scope-btn {
  cursor: pointer; padding: 2px 6px; border-radius: 3px;
  transition: all 0.15s;
}
.scope-btn:hover { background: #eee; }
.scope-btn.active { color: #2080f0; font-weight: 600; background: #e8f4ff; }
.scope-divider { color: #ddd; }
.search-scope-placeholder {
  display: flex; align-items: center; gap: 4px;
  font-size: 12px; flex-shrink: 0;
  visibility: hidden; pointer-events: none;
}
.all-search-results {
  max-height: 240px; overflow-y: auto;
  border-top: 1px solid #eee; padding-top: 8px;
  display: flex; flex-direction: column; gap: 8px;
}
.all-search-chapter-title {
  font-size: 13px; font-weight: 600; color: #333;
  cursor: pointer; padding: 4px 6px; border-radius: 4px;
  margin-bottom: 2px;
}
.all-search-chapter-title:hover { background: #e8f4ff; color: #2080f0; }
.all-search-snippet {
  font-size: 12px; color: #666; cursor: pointer;
  padding: 3px 8px; border-radius: 3px; line-height: 1.5;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.all-search-snippet:hover { background: #f5f5f5; }
.snippet-before, .snippet-after { color: #999; }
.snippet-match { color: #d03050; font-weight: 600; background: #fff0f0; border-radius: 2px; padding: 0 1px; }
.all-search-more { font-size: 11px; color: #999; padding: 2px 8px; }
</style>
