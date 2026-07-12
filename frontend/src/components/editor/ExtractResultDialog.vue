<script setup lang="ts">
import { ref } from 'vue'
import { NModal, NButton, NIcon, NSpace, NTag, useMessage } from 'naive-ui'
import type { Character, Event, ExtractionResult, NovelRelationship } from '../../types'
import { useNovelStore } from '../../stores/novel'

const props = defineProps<{
  show: boolean
  extractResult: ExtractionResult | null
  currentNovelId?: string
  existingOutline?: string
  existingCharacters?: Character[]
  existingRelationships?: NovelRelationship[]
  existingEvents?: Event[]
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  /** 用户应用了变更 */
  applied: []
}>()

const novelStore = useNovelStore()
const message = useMessage()
const applying = ref(false)

// 合并角色列表：按 name 去重，已有角色的补充空字段
function mergeCharacters(existing: Character[], incoming: Character[]): Character[] {
  const map = new Map(existing.map(c => [c.name, { ...c }]))
  for (const ch of incoming) {
    if (map.has(ch.name)) {
      const exist = map.get(ch.name)!
      if (ch.description && !exist.description) exist.description = ch.description
      if (ch.alias && !exist.alias) exist.alias = ch.alias
      if (ch.traits && !exist.traits) exist.traits = ch.traits
    } else {
      map.set(ch.name, { ...ch })
    }
  }
  return Array.from(map.values())
}

// 合并关系列表：按 source+target+relationType 去重
function mergeRelationships(existing: NovelRelationship[], incoming: NovelRelationship[]): NovelRelationship[] {
  const set = new Set(existing.map(r => `${r.source}|${r.target}|${r.relationType}`))
  const merged = [...existing]
  for (const r of incoming) {
    const key = `${r.source}|${r.target}|${r.relationType}`
    if (!set.has(key)) { merged.push(r); set.add(key) }
  }
  return merged
}

// 合并事件列表：按 name 去重，ExtractedEvent → Event 转换
function mergeEvents(existing: Event[], incoming: ExtractionResult['events']): Event[] {
  const set = new Set(existing.map(e => e.name))
  const merged = [...existing]
  for (const e of incoming) {
    if (!set.has(e.name)) {
      merged.push({ name: e.name, description: e.description, timeOrder: e.timeOrder })
      set.add(e.name)
    }
  }
  return merged
}

// 统计新增数量
function countNew(incoming: any[], existing: any[], key: string): number {
  const existSet = new Set(existing.map(e => String(e[key])))
  return incoming.filter(i => !existSet.has(String(i[key]))).length
}
function isNew(val: string, existing: any[], key: string): boolean {
  return !existing.some(e => String(e[key]) === val)
}
function countNewRels(incoming: NovelRelationship[], existing: NovelRelationship[]): number {
  const existSet = new Set(existing.map(r => `${r.source}|${r.target}|${r.relationType}`))
  return incoming.filter(r => !existSet.has(`${r.source}|${r.target}|${r.relationType}`)).length
}
function isNewRel(rel: NovelRelationship, existing: NovelRelationship[]): boolean {
  return !existing.some(r => r.source === rel.source && r.target === rel.target && r.relationType === rel.relationType)
}

/** 应用 AI 提取的元数据到小说 */
async function handleApply() {
  const n = novelStore.currentNovel
  const r = props.extractResult
  if (!n || !r) return
  applying.value = true

  const existingChars = props.existingCharacters || []
  const existingRels = props.existingRelationships || []
  const existingEvts = props.existingEvents || []

  const mergedChars = mergeCharacters(existingChars, r.characters)
  const mergedRels = mergeRelationships(existingRels, r.relationships)
  const mergedEvents = mergeEvents(existingEvts, r.events)
  const mergedOutline = r.outline || n.outline || ''

  const ok = await novelStore.updateNovelMeta(n.id, {
    outline: mergedOutline,
    characters: mergedChars,
    relationships: mergedRels,
    events: mergedEvents,
  })
  applying.value = false
  if (ok) {
    message.success('元数据更新成功')
    emit('update:show', false)
    emit('applied')
  } else {
    message.error('更新失败')
  }
}

/** 忽略提取结果 */
function handleIgnore() {
  emit('update:show', false)
}
</script>

<template>
  <n-modal :show="show" title="AI 提取到新元数据" preset="card"
    style="width: 520px; max-height: 70vh;" :mask-closable="false" draggable
    @update:show="$event === false && emit('update:show', false)">
    <template v-if="extractResult">
      <div class="extract-preview">
        <!-- 大纲 -->
        <div class="extract-section">
          <div class="extract-label">大纲</div>
          <div class="extract-value" :class="{ 'is-new': extractResult.outline && extractResult.outline !== (existingOutline || '') }">
            {{ extractResult.outline || '（无变化）' }}
          </div>
        </div>
        <!-- 新增角色 -->
        <div class="extract-section">
          <div class="extract-label">角色（新增 {{ countNew(extractResult.characters, existingCharacters || [], 'name') }} 个）</div>
          <div v-if="extractResult.characters.length" class="extract-tags">
            <n-tag v-for="ch in extractResult.characters" :key="ch.name" size="small"
              :type="isNew(ch.name, existingCharacters || [], 'name') ? 'success' : 'default'">
              {{ ch.name }}{{ ch.traits ? `（${ch.traits}）` : '' }}
            </n-tag>
          </div>
          <div v-else class="extract-empty">无</div>
        </div>
        <!-- 新增关系 -->
        <div class="extract-section">
          <div class="extract-label">关系（新增 {{ countNewRels(extractResult.relationships, existingRelationships || []) }} 条）</div>
          <div v-if="extractResult.relationships.length" class="extract-rels">
            <div v-for="(rel, i) in extractResult.relationships" :key="i"
              class="extract-rel" :class="{ 'is-new': isNewRel(rel, existingRelationships || []) }">
              {{ rel.source }} → {{ rel.target }}（{{ rel.relationType }}）
            </div>
          </div>
          <div v-else class="extract-empty">无</div>
        </div>
        <!-- 新增事件 -->
        <div class="extract-section">
          <div class="extract-label">事件（新增 {{ countNew(extractResult.events, existingEvents || [], 'name') }} 个）</div>
          <div v-if="extractResult.events.length" class="extract-tags">
            <n-tag v-for="ev in extractResult.events" :key="ev.name" size="small"
              :type="isNew(ev.name, existingEvents || [], 'name') ? 'success' : 'default'">
              {{ ev.name }}
            </n-tag>
          </div>
          <div v-else class="extract-empty">无</div>
        </div>
      </div>
    </template>
    <template #footer>
      <n-space justify="end">
        <n-button quaternary @click="handleIgnore">忽略</n-button>
        <n-button type="primary" :loading="applying" @click="handleApply">应用变更</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.extract-preview {
  display: flex; flex-direction: column; gap: 16px;
  max-height: 50vh; overflow-y: auto;
}
.extract-section {
  display: flex; flex-direction: column; gap: 6px;
}
.extract-label {
  font-size: 13px; font-weight: 600; color: #555;
}
.extract-value {
  font-size: 13px; line-height: 1.6; color: #333;
  padding: 8px 10px; background: #f9f9f9; border-radius: 4px;
  white-space: pre-wrap;
}
.extract-value.is-new {
  background: #f0fdf4; border-left: 3px solid #52c41a;
}
.extract-tags {
  display: flex; flex-wrap: wrap; gap: 6px;
}
.extract-empty {
  font-size: 12px; color: #999;
}
.extract-rels {
  display: flex; flex-direction: column; gap: 4px;
}
.extract-rel {
  font-size: 13px; padding: 4px 8px; background: #f9f9f9; border-radius: 4px;
}
.extract-rel.is-new {
  background: #f0fdf4; border-left: 3px solid #52c41a;
}
</style>
