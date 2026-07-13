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
  AddOutline as AddIcon, TrashOutline as DeleteIcon,
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

// === 图谱 ===
const {
  svgViewBox,
  getNetwork,
  setOnNodeClick,
  ensureGraphBuilt,
  reLayout,
} = useGraphNetwork(containerRef, chars, rels, connectMode, cancelConnect, enterConnectMode)

// 注册节点点击回调（打开编辑弹框）
setOnNodeClick((idx: number) => openEdit(idx))

onUnmounted(() => {
  removeDragListeners()
})
</script>

<template>
  <div class="graph-wrapper">
    <!-- 顶部工具栏 -->
    <div class="graph-toolbar">
      <n-button size="small" type="primary" @click="openAdd">
        <template #icon><n-icon><AddIcon/></n-icon></template>添加角色
      </n-button>
      <n-button size="small" secondary @click="reLayout" :disabled="!getNetwork()">
        重新布局
      </n-button>
      <n-button size="small" :type="connectMode ? 'warning' : 'default' as any" @click="handleConnectClick" :disabled="characters.length === 0">
        连线
      </n-button>
      <n-text v-if="characters.length > 0" depth="3" style="font-size: 13px;">
        {{ characters.length }} 个角色 · 点击节点编辑 · 拖拽连线建关系
      </n-text>
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
      style="width: 520px;" :mask-closable="false" draggable
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
  gap: 12px;
  margin-bottom: 12px;
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
</style>
