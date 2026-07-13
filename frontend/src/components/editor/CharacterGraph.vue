<script setup lang="ts">
/**
 * 角色关系图谱 — 角色编辑 + vis-network 可视化 + 连线建关系
 */
import { ref, computed, nextTick, onUnmounted } from 'vue'
import {
  NButton, NIcon, NModal, NForm, NFormItem, NInput, NSelect,
  NSpace, useMessage, NEmpty, NText,
} from 'naive-ui'
import {
  AddOutline as AddIcon, GitMergeOutline as MergeIcon,
  RefreshOutline as ReloadIcon, LinkOutline as LinkIcon,
  TrashOutline as DeleteIcon,
} from '@vicons/ionicons5'
import type { Character, NovelRelationship } from '../../types'
import { useNovelStore } from '../../stores/novel'
import { useCharacterEdit } from '../../composables/useCharacterEdit'
import { useGraphNetwork } from '../../composables/useGraphNetwork'

const props = defineProps<{
  characters: Character[]
  relationships?: NovelRelationship[]
  /** 提供 novelId 时，角色变更（新增/编辑/删除）会自动持久化到后端 */
  novelId?: string
}>()
const emit = defineEmits<{
  'update:characters': [chars: Character[]]
  'update:relationships': [rels: NovelRelationship[]]
}>()

const message = useMessage()
const novelStore = useNovelStore()
const containerRef = ref<HTMLDivElement>()
const chars = computed(() => props.characters)
const rels = computed(() => props.relationships)

/** 是否有已定义的关系 */
const hasRelationships = computed(() =>
  props.relationships !== undefined && props.relationships.length > 0,
)

/** 有 novelId 时自动将角色和关系数据持久化到后端 */
async function autoSave(chars: Character[], rels?: NovelRelationship[]) {
  if (!props.novelId) return
  const data: Parameters<typeof novelStore.updateNovelMeta>[1] = { characters: chars }
  if (rels !== undefined) data.relationships = rels
  const ok = await novelStore.updateNovelMeta(props.novelId, data)
  if (!ok) message.error('数据保存失败')
}

// === 角色编辑 ===
const {
  showEdit, editName, editAlias, editTraits, editDesc,
  editRelationships, otherCharNames,
  openAdd, openEdit,
  addRelationship, removeRelationship,
  saveCharacter, removeCharacter,
} = useCharacterEdit(
  chars, rels, emit, message, autoSave,
)

// === 连线模式 ===
const connectMode = ref(false)
const connectSrcIdx = ref(-1)
const connectTgtIdx = ref(-1)
const showConnectDialog = ref(false)
const connectType = ref('')
const connectDesc = ref('')
const dragActive = ref(false)
const dragSourceIdx = ref(-1)
const dragLine = ref({ x1: 0, y1: 0, x2: 0, y2: 0 })

function enterConnectMode() {
  connectMode.value = true
  resetConnectState()
  getNetwork()?.setOptions({ interaction: { dragNodes: false } })
  setCanvasPointerEvents('none')
  const el = containerRef.value
  if (el) el.addEventListener('mousedown', onDragMouseDown)
  message.info('连线模式：从节点拖拽连线到目标节点')
}

function exitConnectMode() {
  connectMode.value = false
  resetConnectState()
  removeDragListeners()
  getNetwork()?.setOptions({ interaction: { dragNodes: true } })
}

function resetConnectState() {
  showConnectDialog.value = false
  connectSrcIdx.value = -1
  connectTgtIdx.value = -1
  connectType.value = ''
  connectDesc.value = ''
  dragActive.value = false
  dragSourceIdx.value = -1
}

function cancelConnect() {
  exitConnectMode()
}

function setCanvasPointerEvents(val: 'auto' | 'none') {
  if (!containerRef.value) return
  const canvases = containerRef.value.querySelectorAll('canvas')
  canvases.forEach(c => c.style.pointerEvents = val)
}

function removeDragListeners() {
  setCanvasPointerEvents('auto')
  const el = containerRef.value
  if (el) el.removeEventListener('mousedown', onDragMouseDown)
  document.removeEventListener('mousemove', onDragMouseMove)
  document.removeEventListener('mouseup', onDragMouseUp)
}

function toggleConnectMode() {
  if (connectMode.value) {
    getNetwork()?.setOptions({ interaction: { dragNodes: true } })
    cancelConnect()
  } else {
    enterConnectMode()
  }
}

/** 点击连线按钮：若网络未就绪则强制构建后再进入连线模式 */
function handleConnectClick() {
  if (!getNetwork()) {
    ensureGraphBuilt()
    return
  }
  toggleConnectMode()
}

function onDragMouseDown(e: MouseEvent) {
  const net = getNetwork()
  if (!net) return
  const el = containerRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top
  const idx = net.getNodeAt({ x, y }) as number | undefined
  if (idx === undefined) return

  e.preventDefault()
  dragSourceIdx.value = idx
  dragActive.value = true

  const canvasPos = net.getPosition(idx)
  const domPos = net.canvasToDOM(canvasPos)
  dragLine.value = { x1: domPos.x, y1: domPos.y, x2: x, y2: y }

  document.addEventListener('mousemove', onDragMouseMove)
  document.addEventListener('mouseup', onDragMouseUp)
}

function onDragMouseMove(e: MouseEvent) {
  if (!dragActive.value) return
  e.preventDefault()
  const el = containerRef.value
  if (el) {
    const rect = el.getBoundingClientRect()
    dragLine.value.x2 = e.clientX - rect.left
    dragLine.value.y2 = e.clientY - rect.top
  }
}

function onDragMouseUp(e: MouseEvent) {
  document.removeEventListener('mousemove', onDragMouseMove)
  document.removeEventListener('mouseup', onDragMouseUp)
  if (!dragActive.value) return
  dragActive.value = false

  const el = containerRef.value
  if (el) {
    const rect = el.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    const targetIdx = getNetwork()?.getNodeAt({ x, y }) as number | undefined
    if (targetIdx !== undefined && targetIdx !== dragSourceIdx.value) {
      connectSrcIdx.value = dragSourceIdx.value
      connectTgtIdx.value = targetIdx
      showConnectDialog.value = true
    }
  }
  dragSourceIdx.value = -1
}

function confirmConnect() {
  if (!connectType.value.trim()) {
    message.warning('请输入关系类型')
    return
  }
  const srcName = props.characters[connectSrcIdx.value].name
  const tgtName = props.characters[connectTgtIdx.value].name
  const newRel: NovelRelationship = {
    source: srcName,
    target: tgtName,
    relationType: connectType.value.trim(),
    description: connectDesc.value.trim() || undefined,
  }
  const newRels = [...(props.relationships || []), newRel]
  emit('update:relationships', newRels)
  autoSave(props.characters, newRels)
  message.success(`已添加「${srcName}」→「${tgtName}」`)
  resetConnectState()
}

// === 合并角色 ===
const showMergeDialog = ref(false)
const mergeTargetIdx = ref<number | null>(null)
const mergeSourceIdxs = ref<number[]>([])

/** 角色选择选项（用于合并对话框） */
const charOptions = computed(() =>
  props.characters.map((c, i) => ({ label: c.name, value: i })),
)

function openMergeDialog() {
  mergeTargetIdx.value = null
  mergeSourceIdxs.value = []
  showMergeDialog.value = true
}

function confirmMerge() {
  if (mergeTargetIdx.value === null) {
    message.warning('请选择目标角色')
    return
  }
  if (mergeSourceIdxs.value.length === 0) {
    message.warning('请选择要合并的角色')
    return
  }

  const target = props.characters[mergeTargetIdx.value]
  const targetName = target.name

  // 过滤掉目标自身，避免自合并
  const sources = mergeSourceIdxs.value
    .filter(i => i !== mergeTargetIdx.value)
    .map(i => props.characters[i])
  if (sources.length === 0) {
    message.warning('目标角色不能同时作为被合并角色')
    return
  }

  const sourceNames = sources.map(s => s.name)

  // 收集被合并角色的元数据
  const sourceAliases = sources.map(s => s.alias).filter(Boolean) as string[]
  const sourceTraits = sources.map(s => s.traits).filter(Boolean) as string[]
  const sourceDescs = sources.map(s => s.description).filter(Boolean) as string[]

  // 更新角色列表：保留目标，移除源角色
  const newChars = props.characters.filter(
    (_, i) => i === mergeTargetIdx.value || !mergeSourceIdxs.value.includes(i),
  )

  // 合并元数据到目标角色（逐项去重）
  const targetIdx = newChars.findIndex(c => c.name === targetName)
  if (targetIdx >= 0) {
    const merged = { ...newChars[targetIdx] }
    if (sourceAliases.length) {
      const parts = [merged.alias, ...sourceAliases]
        .filter(Boolean)
        .flatMap(s => (s as string).split(/[、，,]/).map(x => x.trim()).filter(Boolean))
      merged.alias = [...new Set(parts)].join('、')
    }
    if (sourceTraits.length) {
      const parts = [merged.traits, ...sourceTraits]
        .filter(Boolean)
        .flatMap(s => (s as string).split(/[、，,]/).map(x => x.trim()).filter(Boolean))
      merged.traits = [...new Set(parts)].join('、')
    }
    if (sourceDescs.length) {
      // 过滤掉与目标描述完全相同的段落，再按行去重
      const uniqueDescs = sourceDescs.filter(d => d !== merged.description)
      const lines = [merged.description, ...uniqueDescs]
        .filter(Boolean)
        .flatMap(s => (s as string).split('\n').map(x => x.trim()).filter(Boolean))
      merged.description = [...new Set(lines)].join('\n')
    }
    newChars[targetIdx] = merged
  }

  // 重映射关系：将被合并角色的名称替换为目标角色名称
  let newRels = (props.relationships || []).map(r => {
    if (sourceNames.includes(r.source)) r = { ...r, source: targetName }
    if (sourceNames.includes(r.target)) r = { ...r, target: targetName }
    return r
  })

  // 去重：同 source-target-type 合并为一条，排除自引用
  const seen = new Set<string>()
  newRels = newRels.filter(r => {
    if (r.source === r.target) return false // 自引用关系无效
    const key = `${r.source}||${r.target}||${r.relationType}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })

  emit('update:characters', newChars)
  emit('update:relationships', newRels)
  showMergeDialog.value = false
  autoSave(newChars, newRels)
  message.success(`已合并 ${sources.length} 个角色到「${targetName}」`)
}

// === 环状布局 ===
/** 应用环状排列 + 折线连接 */
function applyCircleLayout() {
  const el = containerRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const count = props.characters.length
  if (count < 2) return
  const cx = rect.width / 2
  const cy = rect.height / 2
  const radius = Math.min(cx, cy) - 80
  const positions = props.characters.map((_, i) => {
    const angle = (2 * Math.PI * i) / count - Math.PI / 2
    return { x: cx + radius * Math.cos(angle), y: cy + radius * Math.sin(angle) }
  })
  applyFixedLayout(positions, { type: 'discrete', roundness: 0 })
}

// === 图谱 ===
const {
  svgViewBox,
  getNetwork,
  setOnNodeClick,
  setOnGraphBuilt,
  ensureGraphBuilt,
  reLayout,
  applyFixedLayout,
} = useGraphNetwork(containerRef, chars, rels, connectMode, cancelConnect, enterConnectMode)

// 注册节点点击回调（打开编辑弹框）
setOnNodeClick((idx: number) => openEdit(idx))

// 图重建后自动恢复环状布局
setOnGraphBuilt(() => { nextTick(applyCircleLayout) })

onUnmounted(() => {
  removeDragListeners()
})
</script>

<template>
  <div class="graph-wrapper">
    <!-- 顶部工具栏 -->
    <div class="graph-toolbar">
      <div class="toolbar-actions">
        <n-button class="toolbar-btn" size="small" secondary @click="openAdd">
          <template #icon><n-icon><AddIcon/></n-icon></template>添加角色
        </n-button>
        <n-button class="toolbar-btn" size="small" secondary
          :class="{ active: connectMode }"
          @click="handleConnectClick"
          :disabled="characters.length === 0">
          <template #icon><n-icon><LinkIcon/></n-icon></template>{{ connectMode ? '退出连线' : '连线' }}
        </n-button>
        <n-button class="toolbar-btn" size="small" secondary
          @click="openMergeDialog"
          :disabled="characters.length < 2">
          <template #icon><n-icon><MergeIcon/></n-icon></template>合并
        </n-button>
        <n-button class="toolbar-btn" size="small" secondary
          @click="applyCircleLayout"
          :disabled="!getNetwork()">
          <template #icon><n-icon><ReloadIcon/></n-icon></template>重新布局
        </n-button>
      </div>
      <div class="toolbar-info">
        <n-text v-if="characters.length > 0" depth="3" style="font-size: 13px;">
          {{ characters.length }} 个角色
        </n-text>
      </div>
    </div>

    <!-- 连线模式横幅 -->
    <div v-if="connectMode" class="connect-banner">
      <span>🔗 连线模式 — 从节点拖拽到目标角色建立关系</span>
      <n-button size="tiny" text style="color: #fff; text-decoration: underline;" @click="cancelConnect">退出连线</n-button>
    </div>

    <!-- vis-network 画布 + SVG 拖拽叠加层 -->
    <div v-if="characters.length > 0" class="graph-area">
      <div ref="containerRef" class="graph-container" :class="{ 'connect-mode': connectMode }" />
      <svg class="graph-svg-overlay" :viewBox="svgViewBox">
        <defs>
          <marker id="drag-arrow" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
            <polygon points="0 0, 10 3.5, 0 7" fill="#2080f0" />
          </marker>
        </defs>
        <line v-if="dragActive"
          :x1="dragLine.x1" :y1="dragLine.y1"
          :x2="dragLine.x2" :y2="dragLine.y2"
          stroke="#2080f0" stroke-width="2.5"
          stroke-dasharray="6, 4" stroke-linecap="round"
          marker-end="url(#drag-arrow)" />
      </svg>
    </div>
    <div v-if="characters.length > 0 && !hasRelationships" class="graph-hint">
      角色已就绪，点击「连线」按钮拖拽建立关系
    </div>
    <n-empty v-if="characters.length === 0" description="还没有角色" class="graph-empty">
      <template #extra>
        <n-button size="small" @click="openAdd">添加第一个角色</n-button>
      </template>
    </n-empty>

    <!-- 角色编辑弹框 -->
    <n-modal class="dialog-modal" :show="showEdit" title="编辑角色" preset="card"
      style="width: 520px; max-height: 88vh;" :mask-closable="false" draggable
      @update:show="showEdit = $event">
      <n-form label-placement="top">
        <n-form-item label="角色名称" required>
          <n-input v-model:value="editName" placeholder="输入角色名称" />
        </n-form-item>
        <n-form-item label="别名">
          <n-input v-model:value="editAlias" placeholder="别名 / 绰号" />
        </n-form-item>
        <n-form-item label="性格特征">
          <n-input v-model:value="editTraits" placeholder="例：勇敢、智慧、固执" />
        </n-form-item>
        <n-form-item label="描述">
          <n-input v-model:value="editDesc" type="textarea" :rows="2" placeholder="角色详细描述" />
        </n-form-item>
        <n-form-item label="与其他角色的关系">
          <div class="relationship-list">
            <div v-for="(rel, ri) in editRelationships" :key="ri" class="relationship-row">
              <n-select
                v-model:value="rel.targetName"
                :options="otherCharNames.map(n => ({ label: n, value: n }))"
                placeholder="选择角色" filterable style="width: 130px;" size="small" />
              <n-button text size="small" style="width: 28px; font-size: 16px; flex-shrink: 0;"
                @click="rel.isIncoming = !rel.isIncoming"
                :title="rel.isIncoming ? '入向（←）' : '出向（→）'">
                {{ rel.isIncoming ? '←' : '→' }}
              </n-button>
              <n-input v-model:value="rel.relationType" placeholder="关系类型" size="small" style="width: 100px;" />
              <n-input v-model:value="rel.description" placeholder="描述" size="small" style="flex: 1; min-width: 0;" />
              <n-button text size="small" style="color: #d03050; flex-shrink: 0;" @click="removeRelationship(ri)">
                <template #icon><n-icon size="16"><DeleteIcon/></n-icon></template>
              </n-button>
            </div>
            <n-button size="tiny" quaternary @click="addRelationship">
              <template #icon><n-icon><AddIcon/></n-icon></template>添加关系
            </n-button>
          </div>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showEdit = false">取消</n-button>
          <n-button type="primary" @click="saveCharacter">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 连线模式弹框 -->
    <n-modal :show="showConnectDialog" title="添加关系" preset="card"
      style="width: 360px;" :mask-closable="false"
      @update:show="showConnectDialog = $event">
      <n-form label-placement="top">
        <n-form-item label="关系类型" required>
          <n-input v-model:value="connectType" placeholder="例：朋友、敌人、恋人" />
        </n-form-item>
        <n-form-item label="描述（可选）">
          <n-input v-model:value="connectDesc" placeholder="关系描述" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="cancelConnect">取消</n-button>
          <n-button type="primary" @click="confirmConnect">确定</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 合并角色弹框 -->
    <n-modal :show="showMergeDialog" title="合并角色" preset="card"
      style="width: 480px;" :mask-closable="false" draggable
      @update:show="showMergeDialog = $event">
      <n-form label-placement="top">
        <n-form-item label="目标角色（合并到该角色）" required>
          <n-select v-model:value="mergeTargetIdx" :options="charOptions" placeholder="选择保留的角色" filterable />
        </n-form-item>
        <n-form-item label="被合并角色（将被移除）" required>
          <n-select
            v-model:value="mergeSourceIdxs"
            :options="charOptions"
            placeholder="选择要合并进来的角色"
            multiple filterable
            :disabled="mergeTargetIdx === null"
          />
        </n-form-item>
        <n-text depth="3" style="font-size: 12px;">
          提示：被合并角色的别名、特征和描述会合并到目标角色，涉及的关系会自动重映射
        </n-text>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button quaternary @click="showMergeDialog = false">取消</n-button>
          <n-button type="primary" @click="confirmMerge">确认合并</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style lang="scss" scoped>
.graph-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.graph-toolbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-btn { --n-border-color: #d9d9d9; }

.toolbar-btn.active {
  --n-text-color: #2080f0 !important;
  --n-border-color: #2080f0 !important;
  --n-color: #ecf5ff !important;
}

.toolbar-info { flex-shrink: 0; }

/* 连线模式横幅 */
.connect-banner {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 14px;
  margin-bottom: 10px;
  background: #2080f0;
  color: #fff;
  border-radius: 6px;
  font-size: 13px;
}

/* vis-network 画布容器 */
.graph-area {
  flex: 1;
  min-height: 0;
  position: relative;
  display: flex;
  flex-direction: column;
}
.graph-container {
  flex: 1;
  min-height: 0;
  position: relative;
  border: 1px solid #eee;
  border-radius: 6px;
  background: #fafafa;
  overflow: hidden;

  &.connect-mode { cursor: crosshair; }
}

/* SVG 拖拽线叠加层 */
.graph-svg-overlay {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 10;
}
/* 无关系提示 */
.graph-hint {
  flex-shrink: 0;
  padding: 6px 12px;
  background: #f0f5ff;
  border-radius: 4px;
  font-size: 12px;
  color: #888;
  text-align: center;
}

.graph-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 关系列表 */
.relationship-list {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.relationship-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

/* 弹框内容溢出滚动 */
.dialog-modal :deep(.n-card__content) {
  overflow-y: auto;
}
</style>
