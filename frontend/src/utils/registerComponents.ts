/**
 * 全局注册异步组件（弹框类）
 * 按需加载，避免主 chunk 体积过大，也省去各文件重复 import
 */
import type { App } from 'vue'
import { defineAsyncComponent } from 'vue'

/** 编辑器弹框 */
const SplitChapterDialog = defineAsyncComponent(() =>
  import('../components/editor/SplitChapterDialog.vue'))
const AIContinueDialog = defineAsyncComponent(() =>
  import('../components/editor/AIContinueDialog.vue'))
const AIEditDialog = defineAsyncComponent(() =>
  import('../components/editor/AIEditDialog.vue'))
const AISetupDialog = defineAsyncComponent(() =>
  import('../components/editor/AISetupDialog.vue'))
const NovelInfoDialog = defineAsyncComponent(() =>
  import('../components/editor/NovelInfoDialog.vue'))
const ExtractResultDialog = defineAsyncComponent(() =>
  import('../components/editor/ExtractResultDialog.vue'))

/** 小说列表弹框 */
const CreateNovelDialog = defineAsyncComponent(() =>
  import('../components/novel/CreateNovelDialog.vue'))
const ImportNovelDialog = defineAsyncComponent(() =>
  import('../components/novel/ImportNovelDialog.vue'))
const RenameNovelDialog = defineAsyncComponent(() =>
  import('../components/novel/RenameNovelDialog.vue'))
const DeleteNovelConfirmDialog = defineAsyncComponent(() =>
  import('../components/novel/DeleteNovelConfirmDialog.vue'))

/** 注册全部弹框为全局组件，模板中直接使用 PascalCase 标签名 */
export function registerGlobalComponents(app: App) {
  app.component('SplitChapterDialog', SplitChapterDialog)
  app.component('AIContinueDialog', AIContinueDialog)
  app.component('AIEditDialog', AIEditDialog)
  app.component('AISetupDialog', AISetupDialog)
  app.component('NovelInfoDialog', NovelInfoDialog)
  app.component('ExtractResultDialog', ExtractResultDialog)
  app.component('CreateNovelDialog', CreateNovelDialog)
  app.component('ImportNovelDialog', ImportNovelDialog)
  app.component('RenameNovelDialog', RenameNovelDialog)
  app.component('DeleteNovelConfirmDialog', DeleteNovelConfirmDialog)
}
