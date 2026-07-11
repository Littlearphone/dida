import { computed, ref, watch } from 'vue'
import { useSettingsStore } from '../stores/settings'

/** 编辑器外观状态（字体、行距、段落间距等） */
export function useEditorAppearance() {
  const settingsStore = useSettingsStore()

  const fontSize = ref(16)
  const lineHeight = ref(1.8)
  const paragraphSpacing = ref(16)
  const isBold = ref(false)
  const isItalic = ref(false)
  const fontFamily = ref('')

  const fontOptions = [
    { label: '系统默认', value: '' },
    { label: '宋体', value: 'SimSun, serif' },
    { label: '黑体', value: 'SimHei, sans-serif' },
    { label: '微软雅黑', value: '"Microsoft YaHei", sans-serif' },
    { label: '楷体', value: 'KaiTi, serif' },
    { label: '仿宋', value: 'FangSong, serif' },
  ]

  /** 供 editor-content div 使用的 :style 绑定 */
  const editorStyles = computed(() => ({
    fontSize: fontSize.value + 'px',
    lineHeight: lineHeight.value,
    fontWeight: isBold.value ? 'bold' : 'normal',
    fontStyle: isItalic.value ? 'italic' : 'normal',
    fontFamily: fontFamily.value || undefined,
    '--p-gap': paragraphSpacing.value + 'px',
  }))

  /** 从设置中初始化外观参数 */
  function initFromSettings() {
    if (settingsStore.settings) {
      fontSize.value = settingsStore.settings.defaultFontSize || 16
      lineHeight.value = settingsStore.settings.defaultLineSpacing || 1.8
    }
  }

  // 设置变更时自动同步
  watch(() => settingsStore.settings, (s) => {
    if (s) {
      fontSize.value = s.defaultFontSize || 16
      lineHeight.value = s.defaultLineSpacing || 1.8
    }
  })

  return {
    fontSize,
    lineHeight,
    paragraphSpacing,
    isBold,
    isItalic,
    fontFamily,
    fontOptions,
    editorStyles,
    initFromSettings,
  }
}
