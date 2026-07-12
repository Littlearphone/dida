<script setup lang="ts">
import { NButton, NDivider, NIcon, NSelect, NText, NTooltip } from 'naive-ui'
import {
  ArrowUndoOutline as UndoIcon, ArrowRedoOutline as RedoIcon,
  SearchOutline as SearchIcon,
  HammerOutline as FormatIcon, SaveOutline as SaveIcon,
} from '@vicons/ionicons5'

defineProps<{
  undoable: boolean; redoable: boolean
  fontSize: number; lineHeight: number; paragraphSpacing: number
  isBold: boolean; isItalic: boolean; fontFamily: string
  showSearch: boolean
  canFormat: boolean
  fontOptions: { label: string; value: string }[]
  contentChanged: boolean
  autoSaveEnabled: boolean
}>()

const emit = defineEmits<{
  undo: []; redo: []
  'update:fontSize': [v: number]; 'update:lineHeight': [v: number]
  'update:paragraphSpacing': [v: number]
  'update:isBold': [v: boolean]; 'update:isItalic': [v: boolean]
  'update:fontFamily': [v: string]
  'update:showSearch': [v: boolean]
  formatContent: []; save: []
}>()
</script>

<template>
  <div class="toolbar">
    <div class="toolbar-inner">
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" :disabled="!undoable" @click="emit('undo')">
            <template #icon><n-icon size="16"><UndoIcon/></n-icon></template>
          </n-button>
        </template>
        撤销 (Ctrl+Z)
      </n-tooltip>
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" :disabled="!redoable" @click="emit('redo')">
            <template #icon><n-icon size="16"><RedoIcon/></n-icon></template>
          </n-button>
        </template>
        重做 (Ctrl+Shift+Z)
      </n-tooltip>
      <n-divider vertical />
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" :type="isBold ? 'primary' : 'default'" @click="emit('update:isBold', !isBold)">
            <template #icon><span class="bi-icon">B</span></template>
          </n-button>
        </template>
        全局粗体 (Ctrl+B)
      </n-tooltip>
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" :type="isItalic ? 'primary' : 'default'" @click="emit('update:isItalic', !isItalic)">
            <template #icon><span class="bi-icon italic">I</span></template>
          </n-button>
        </template>
        全局斜体 (Ctrl+I)
      </n-tooltip>
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" @click="emit('update:showSearch', !showSearch)">
            <template #icon><n-icon size="16"><SearchIcon/></n-icon></template>
          </n-button>
        </template>
        搜索 (Ctrl+F)
      </n-tooltip>
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" :disabled="!canFormat" @click="emit('formatContent')">
            <template #icon><n-icon size="16"><FormatIcon/></n-icon></template>
          </n-button>
        </template>
        删除多余空行 (Ctrl+Shift+F)
      </n-tooltip>
      <n-divider vertical />
      <n-select :value="fontFamily" :options="fontOptions" size="small" style="width: 112px;" placeholder="字体"
        @update:value="(v: string) => emit('update:fontFamily', v)" />
      <n-divider vertical />
      <!-- 字号：小 A / 大 A 风格 -->
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" @click="emit('update:fontSize', Math.max(12, fontSize - 1))">
            <template #icon><span class="a-wrap"><span class="a-sm">A</span><span class="a-sign">−</span></span></template>
          </n-button>
        </template>
        缩小字号
      </n-tooltip>
      <n-text class="ctrl-value">{{ fontSize }}</n-text>
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" @click="emit('update:fontSize', Math.min(32, fontSize + 1))">
            <template #icon><span class="a-wrap"><span class="a-lg">A</span><span class="a-sign plus">+</span></span></template>
          </n-button>
        </template>
        增大字号
      </n-tooltip>
      <n-divider vertical />
      <!-- 行距：间距压缩 / 拉伸图标 -->
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" @click="emit('update:lineHeight', Math.round(Math.max(1, lineHeight - 0.1) * 10) / 10)">
            <template #icon>
              <span class="line-icon">
                <span class="l1"></span><span class="l2"></span><span class="l2"></span>
              </span>
            </template>
          </n-button>
        </template>
        减小行距
      </n-tooltip>
      <n-text class="ctrl-value">{{ lineHeight.toFixed(1) }}</n-text>
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" @click="emit('update:lineHeight', Math.min(3, +(lineHeight + 0.1).toFixed(1)))">
            <template #icon>
              <span class="line-icon wide">
                <span class="l1"></span><span class="l2"></span><span class="l2"></span>
              </span>
            </template>
          </n-button>
        </template>
        增大行距
      </n-tooltip>
      <n-divider vertical />
      <!-- 段距：段落间距收窄 / 拉伸 -->
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" @click="emit('update:paragraphSpacing', Math.max(0, paragraphSpacing - 4))">
            <template #icon><span class="para-icon narrow">¶¶</span></template>
          </n-button>
        </template>
        减小段距
      </n-tooltip>
      <n-text class="ctrl-value">{{ paragraphSpacing }}</n-text>
      <n-tooltip trigger="hover" placement="bottom">
        <template #trigger>
          <n-button quaternary size="tiny" @click="emit('update:paragraphSpacing', Math.min(48, paragraphSpacing + 4))">
            <template #icon><span class="para-icon wide">¶ ¶</span></template>
          </n-button>
        </template>
        增大段距
      </n-tooltip>
      <template v-if="!autoSaveEnabled">
        <n-divider vertical />
        <n-tooltip trigger="hover" placement="bottom">
          <template #trigger>
            <n-button type="warning" size="tiny" :disabled="!contentChanged" @click="emit('save')">
              <template #icon><n-icon size="16"><SaveIcon/></n-icon></template>
            </n-button>
          </template>
          保存
        </n-tooltip>
      </template>
    </div>
  </div>
</template>

<style scoped>
.toolbar {
  border-bottom: 1px solid #eee;
  padding: 4px 8px;
  display: flex;
  align-items: center;
  flex-shrink: 0;
}
.toolbar-inner {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: wrap;
  justify-content: center;
  min-width: 0;
}
/* 缩小 NaiveUI 垂直分隔线的间距 */
:deep(.n-divider--vertical) {
  margin: 0 1px !important;
  width: 1px !important;
}
.ctrl-value {
  font-size: 12px;
  min-width: 24px;
  text-align: center;
  white-space: nowrap;
}
.bi-icon {
  display: inline-flex; align-items: center; justify-content: center;
  width: 16px; height: 16px;
  font-weight: 400; font-size: 15px; line-height: 1;
}
.bi-icon.italic { font-style: italic; }

/* 字号：小 A / 大 A（右上角带 +/-） */
.a-wrap {
  position: relative;
  display: inline-flex;
  align-items: flex-end;
  justify-content: center;
  width: 16px; height: 16px;
}
.a-sm, .a-lg {
  font-weight: 700; line-height: 1;
}
.a-sm { font-size: 14px; }
.a-lg { font-size: 17px; }
.a-sign {
  position: absolute;
  top: -2px; right: -5px;
  font-size: 10px;
  font-weight: 900;
  line-height: 1;
  color: inherit;
  opacity: 0.8;
}
.a-sign.plus { top: -3px; }

/* 行距：三条线表示 */
.line-icon {
  display: flex; flex-direction: column; align-items: center;
  width: 16px; gap: 2px;
}
.line-icon span {
  display: block; width: 14px; height: 2px; background: currentColor; border-radius: 1px;
}
.line-icon .l2 { width: 10px; }
.line-icon.wide { gap: 3px; }

/* 段距：¶ 符号间距变化 */
.para-icon { font-size: 11px; line-height: 1; }
.para-icon.narrow { letter-spacing: -1px; }
.para-icon.wide { letter-spacing: 0; }
</style>
