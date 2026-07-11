<script setup lang="ts">
import type {Novel} from '../types'
import {NCard, NIcon, NText} from 'naive-ui'
import {
  CreateOutline as RenameIcon,
  DocumentTextOutline as DescIcon,
  TrashOutline as DeleteIcon,
} from '@vicons/ionicons5'

const props = defineProps<{
  novel: Novel
}>()

const emit = defineEmits<{
  click: [novel: Novel]
  rename: [novel: Novel]
  editDesc: [novel: Novel]
  delete: [novel: Novel]
}>()

/** 基于小说ID生成封面主色 */
function getCoverColor(id: string): string {
  const colors = [
    '#c0392b', '#2980b9', '#27ae60', '#8e44ad',
    '#d35400', '#16a085', '#d4ac0d', '#2c3e50',
  ]
  const hash = id.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0)
  return colors[hash % colors.length]
}

/** 生成封面渐变色（更深色用于书脊） */
function getCoverGradient(id: string): string {
  const base = getCoverColor(id)
  // 加深20%用于书脊
  return `linear-gradient(to right, ${darkenColor(base, 0.35)} 6px, ${base} 6px, ${lightenColor(base, 0.15)} 100%)`
}

function darkenColor(hex: string, factor: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgb(${Math.round(r * (1 - factor))},${Math.round(g * (1 - factor))},${Math.round(b * (1 - factor))})`
}

function lightenColor(hex: string, factor: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgb(${Math.min(255, Math.round(r + (255 - r) * factor))},${Math.min(255, Math.round(g + (255 - g) * factor))},${Math.min(255, Math.round(b + (255 - b) * factor))})`
}

/** 删除按钮阻止冒泡 */
function handleDelete(e: MouseEvent) {
  e.stopPropagation()
  emit('delete', props.novel)
}
function handleRename(e: MouseEvent) {
  e.stopPropagation()
  emit('rename', props.novel)
}
function handleEditDesc(e: MouseEvent) {
  e.stopPropagation()
  emit('editDesc', props.novel)
}
</script>

<template>
  <n-card
    class="novel-card"
    hoverable
    :bordered="true"
    @click="emit('click', novel)"
  >
    <div class="cover-wrapper">
      <div
        class="book-cover"
        :style="{ background: getCoverGradient(novel.id) }"
      >
        <!-- 书脊装饰线 -->
        <div class="spine-line" />
        <!-- 封面标题 -->
        <div class="cover-content">
          <n-text class="cover-title" :title="novel.title">
            {{ novel.title }}
          </n-text>
          <n-text v-if="novel.author" class="cover-author">
            {{ novel.author }}
          </n-text>
        </div>
      </div>
    </div>

    <!-- 书籍信息（标题已在封面，不再重复） -->
    <div class="book-info">
      <n-text
        v-if="novel.description"
        class="book-desc"
        depth="3"
        :title="novel.description"
      >
        {{ novel.description.slice(0, 40) }}{{ novel.description.length > 40 ? '...' : '' }}
      </n-text>
      <n-text class="book-meta" depth="3">
        {{ novel.wordCount.toLocaleString() }} 字 · {{ novel.chapterIds.length }} 章
      </n-text>
    </div>

    <!-- Hover 操作栏（卡片底部，等宽水平按钮） -->
    <div class="action-bar">
      <button class="action-btn" @click="handleRename">
        <n-icon size="15"><RenameIcon /></n-icon>
        <span>重命名</span>
      </button>
      <button class="action-btn" @click="handleEditDesc">
        <n-icon size="15"><DescIcon /></n-icon>
        <span>简介</span>
      </button>
      <button class="action-btn action-btn-danger" @click="handleDelete">
        <n-icon size="15"><DeleteIcon /></n-icon>
        <span>删除</span>
      </button>
    </div>
  </n-card>
</template>

<style scoped>
.novel-card {
  width: 180px;
  position: relative;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
  border-radius: 8px;
  overflow: hidden; /* 防止上浮边框溢出可视范围 */
  --n-padding: 10px !important;

  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  }

  .cover-wrapper {
    position: relative;
    margin-bottom: 8px;
  }

  .book-cover {
    position: relative;
    width: 100%;
    height: 180px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
  }

  .spine-line {
    position: absolute;
    left: 3px;
    top: 8px;
    bottom: 8px;
    width: 1px;
    background: rgba(255, 255, 255, 0.15);
    border-radius: 1px;
  }

  .cover-content {
    position: relative;
    padding: 20px 16px 16px 20px;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    max-width: 100%;
  }

  .cover-title {
    color: rgba(255, 255, 255, 0.95);
    font-size: 16px;
    font-weight: 600;
    line-height: 1.4;
    text-shadow: 0 1px 4px rgba(0, 0, 0, 0.2);
    display: -webkit-box;
    -webkit-line-clamp: 4;
    -webkit-box-orient: vertical;
    overflow: hidden;
    word-break: break-word;
  }

  .cover-author {
    color: rgba(255, 255, 255, 0.7);
    font-size: 12px;
  }

  .book-info {
    text-align: center;
    transition: margin-bottom 0.2s ease;
  }

  &:hover .book-info {
    margin-bottom: 24px; /* 给底部滑入的操作栏腾空间，不遮挡文字 */
  }

  .book-desc {
    display: block;
    font-size: 11px;
    margin-bottom: 3px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    line-height: 1.4;
  }

  .book-meta {
    font-size: 11px;
  }

  /* 卡片从下方溢出部分被 overflow:hidden 裁切 */
  /* Hover 操作栏 — 从卡片下方滑入，不遮挡文字 */
  .action-bar {
    position: absolute;
    bottom: -34px;
    left: 0;
    right: 0;
    display: flex;
    flex-flow: nowrap;
    background: #fff;
    border-top: 1px solid #e8e8e8;
    transition: bottom 0.2s ease;
    overflow: hidden;
  }

  &:hover .action-bar {
    bottom: 0;
  }

  .action-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 3px;
    height: 34px;
    border: none;
    background: transparent;
    color: #555;
    font-size: 12px;
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
    &:not(:last-child) { border-right: 1px solid #eee; }
    &:hover { background: #f5f5f5; color: #333; }
  }

  .action-btn span {
    max-width: 0;
    overflow: hidden;
    transition: max-width 0.2s ease;
    white-space: nowrap;
    display: inline-block;
  }

  .action-btn:hover span {
    max-width: 60px;
  }

  .action-btn-danger:hover { color: #d03050 !important; background: #fff5f5 !important; }
}
</style>
