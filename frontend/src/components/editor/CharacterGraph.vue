<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import {
  NButton, NIcon, NModal, NForm, NFormItem, NInput, NSelect,
  NSpace, useMessage, NEmpty, NText,
} from 'naive-ui'
import {
  AddOutline as AddIcon, TrashOutline as DeleteIcon,
} from '@vicons/ionicons5'
import { Network } from 'vis-network'
import { DataSet } from 'vis-data'
import type { Character, NovelRelationship } from '../../types'
import { useNovelStore } from '../../stores/novel'
import { buildNodes, buildEdges, DEFAULT_GRAPH_OPTIONS } from './graphUtils'

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
let network: any = null

/** 是否有已定义的关系 */
const hasRelationships = computed(() =>
  props.relationships !== undefined && props.relationships.length > 0,
)

// ------ 角色编辑弹框 ------
const showEdit = ref(false)
const editingIndex = ref(-1)
const editName = ref('')
const editAlias = ref('')
const editTraits = ref('')
const editDesc = ref('')
/** 编辑弹框中当前角色的关系列表（含方向：isIncoming=true 表示其他角色→当前角色） */
interface EditRel { targetName: string; relationType: string; description?: string; isIncoming?: boolean }
const editRelationships = ref<EditRel[]>([])

// ------ 连线模式（直接在图上点节点建关系）------
const connectMode = ref(false)
const connectSrcIdx = ref(-1)
const connectTgtIdx = ref(-1)
const showConnectDialog = ref(false)
const connectType = ref('')
const connectDesc = ref('')

/** 拖拽绘制连线状态 */
const dragActive = ref(false)
const dragSourceIdx = ref(-1)
const dragLine = ref({ x1: 0, y1: 0, x2: 0, y2: 0 })
/** SVG viewBox，同步到容器实际尺寸，让坐标系 = 像素坐标 */
const svgViewBox = ref('0 0 100 100')

function updateSvgViewBox() {
  if (!containerRef.value) return
  const rect = containerRef.value.getBoundingClientRect()
  svgViewBox.value = `0 0 ${Math.max(1, Math.round(rect.width))} ${Math.max(1, Math.round(rect.height))}`
}

function toggleConnectMode() {
  if (connectMode.value) {
    network?.setOptions({ interaction: { dragNodes: true } })
    cancelConnect()
  } else {
    enterConnectMode()
  }
}

/** 进入连线模式 */
function enterConnectMode() {
  connectMode.value = true
  resetConnectState()
  network?.setOptions({ interaction: { dragNodes: false } })
  setCanvasPointerEvents('none')
  const el = containerRef.value
  if (el) el.addEventListener('mousedown', onDragMouseDown)
  message.info('连线模式：从节点拖拽连线到目标节点')
}

/** 退出连线模式 */
function exitConnectMode() {
  connectMode.value = false
  resetConnectState()
  removeDragListeners()
  network?.setOptions({ interaction: { dragNodes: true } })
}

/** 重置连线状态但不退出连线模式（便于连续建关系） */
function resetConnectState() {
  showConnectDialog.value = false
  connectSrcIdx.value = -1
  connectTgtIdx.value = -1
  connectType.value = ''
  connectDesc.value = ''
  dragActive.value = false
  dragSourceIdx.value = -1
}

/** 启用/禁用 vis-network canvas 的指针事件 */
function setCanvasPointerEvents(val: 'auto' | 'none') {
  if (!containerRef.value) return
  const canvases = containerRef.value.querySelectorAll('canvas')
  canvases.forEach(c => c.style.pointerEvents = val)
}

/** 清理拖拽相关的事件监听 */
function removeDragListeners() {
  setCanvasPointerEvents('auto')
  const el = containerRef.value
  if (el) el.removeEventListener('mousedown', onDragMouseDown)
  document.removeEventListener('mousemove', onDragMouseMove)
  document.removeEventListener('mouseup', onDragMouseUp)
}

function cancelConnect() {
  exitConnectMode()
}

/** 点击连线按钮：若网络未就绪则强制构建后再进入连线模式 */
function handleConnectClick() {
  if (!network) {
    ensureGraphBuilt()
    return
  }
  toggleConnectMode()
}

/** mousedown：检测节点并启动拖拽 */
function onDragMouseDown(e: MouseEvent) {
  if (!network) return
  const el = containerRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top
  const idx = network.getNodeAt({ x, y }) as number | undefined
  if (idx === undefined) return

  e.preventDefault()
  dragSourceIdx.value = idx
  dragActive.value = true

  // 获取源节点中心在 DOM 中的坐标作为线起点
  const canvasPos = network.getPosition(idx)
  const domPos = network.canvasToDOM(canvasPos)
  dragLine.value = { x1: domPos.x, y1: domPos.y, x2: x, y2: y }

  // 在 document 上监听 move/up，防止鼠标移出画布后丢失事件
  document.addEventListener('mousemove', onDragMouseMove)
  document.addEventListener('mouseup', onDragMouseUp)
}

/** 拖拽中：更新 SVG 线的终点 */
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

/** 拖拽结束：检测目标节点 */
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
    const targetIdx = network?.getNodeAt({ x, y }) as number | undefined
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
  // 添加到平铺关系列表
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

/** 其他角色名列表（用于关系下拉选择） */
const otherCharNames = ref<string[]>([])

function openAdd() {
  editingIndex.value = -1
  editName.value = ''
  editAlias.value = ''
  editTraits.value = ''
  editDesc.value = ''
  editRelationships.value = []
  otherCharNames.value = props.characters.map(c => c.name)
  showEdit.value = true
}

function openEdit(index: number) {
  const ch = props.characters[index]
  editingIndex.value = index
  editName.value = ch.name
  editAlias.value = ch.alias || ''
  editTraits.value = ch.traits || ''
  editDesc.value = ch.description || ''
  // 显示所有涉及该角色的关系（出向 + 入向），isIncoming 标记方向
  editRelationships.value = (props.relationships || [])
    .filter(r => r.source === ch.name || r.target === ch.name)
    .map(r => ({ targetName: r.source === ch.name ? r.target : r.source, relationType: r.relationType, description: r.description, isIncoming: r.target === ch.name }))
  // 排除自身
  otherCharNames.value = props.characters
    .filter((_, i) => i !== index)
    .map(c => c.name)
  showEdit.value = true
}

function addRelationship() {
  editRelationships.value.push({ targetName: '', relationType: '', description: '' })
}

function removeRelationship(index: number) {
  editRelationships.value.splice(index, 1)
}

function saveCharacter() {
  if (!editName.value.trim()) {
    message.warning('请输入角色名称')
    return
  }
  // 同名检测
  const dup = props.characters.findIndex(
    (c, i) => c.name === editName.value.trim() && i !== editingIndex.value,
  )
  if (dup >= 0) {
    message.warning('角色名称已存在')
    return
  }
  const charName = editName.value.trim()
  const list = [...props.characters]
  const ch: Character = {
    name: charName,
    alias: editAlias.value.trim() || undefined,
    traits: editTraits.value.trim() || undefined,
    description: editDesc.value.trim() || undefined,
  }
  if (editingIndex.value >= 0) {
    list[editingIndex.value] = ch
  } else {
    list.push(ch)
  }

  // 移除此角色相关的所有旧关系（含作为 source 和 target），再按编辑后的方向写回
  const oldName = editingIndex.value >= 0 ? props.characters[editingIndex.value].name : ''
  let newRels = (props.relationships || []).filter(r => r.source !== oldName && r.target !== oldName)
  const validRels = editRelationships.value.filter(r => r.targetName.trim())
  for (const r of validRels) {
    if (r.isIncoming) {
      // 其他角色 → 当前角色
      newRels.push({ source: r.targetName.trim(), target: charName, relationType: r.relationType.trim(), description: r.description?.trim() || undefined })
    } else {
      // 当前角色 → 其他角色
      newRels.push({ source: charName, target: r.targetName.trim(), relationType: r.relationType.trim(), description: r.description?.trim() || undefined })
    }
  }

  emit('update:characters', list)
  emit('update:relationships', newRels)
  showEdit.value = false
  autoSave(list, newRels)
}

function removeCharacter(index: number) {
  const ch = props.characters[index]
  // 检查是否有涉及该角色的关系
  const involvedRels = (props.relationships || []).filter(r => r.source === ch.name || r.target === ch.name)
  if (involvedRels.length > 0) {
    if (!window.confirm(`角色「${ch.name}」存在 ${involvedRels.length} 条关系记录。删除后这些关系将被清理，是否继续？`)) {
      return
    }
  }
  const list = [...props.characters]
  list.splice(index, 1)
  // 清理涉及该角色的所有关系
  const newRels = (props.relationships || []).filter(r => r.source !== ch.name && r.target !== ch.name)

  emit('update:characters', list)
  emit('update:relationships', newRels)
  autoSave(list, newRels)
}

/** 有 novelId 时自动将角色和关系数据持久化到后端 */
async function autoSave(chars: Character[], rels?: NovelRelationship[]) {
  if (!props.novelId) return
  const data: Parameters<typeof novelStore.updateNovelMeta>[1] = { characters: chars }
  if (rels !== undefined) data.relationships = rels
  const ok = await novelStore.updateNovelMeta(props.novelId, data)
  if (!ok) message.error('数据保存失败')
}

// ------ vis-network ------
/** 容器尺寸观察器：等 Tab 切过来容器有真实尺寸后再构建，构建后继续监听容器高度变化重新居中 */
let resizeObserver: ResizeObserver | null = null
let graphBuilt = false
let resizeFitTimer: ReturnType<typeof setTimeout> | null = null

function ensureGraphBuilt() {
  if (graphBuilt || !containerRef.value) return
  const rect = containerRef.value.getBoundingClientRect()
  if (rect.width === 0 || rect.height === 0) return // 容器还不可见，等 ResizeObserver
  graphBuilt = true
  // 不断开 ResizeObserver —— 容器高度可能逐渐增大，需持续监听并重新居中
  buildGraph()
}

/** 容器尺寸变化时防抖重新居中 */
function scheduleResizeFit() {
  if (!network) return
  if (resizeFitTimer) clearTimeout(resizeFitTimer)
  resizeFitTimer = setTimeout(() => {
    network?.fit({ animation: false })
    updateSvgViewBox()
  }, 150)
}

function buildGraph() {
  if (!containerRef.value) return

  // 记住重建前是否处于连线模式（以便重建后恢复，实现连续建关系）
  const wasConnect = connectMode.value

  // 重建时退出连线模式
  cancelConnect()

  // 清理旧实例
  if (network) {
    network.destroy()
    network = null
  }

  const chars = props.characters
  if (chars.length === 0) return

  const nodeItems = buildNodes(chars)
  const edgeItems = buildEdges(chars, props.relationships)

  const nodes = new DataSet(nodeItems as any)
  const edges = new DataSet(edgeItems as any)

  network = new (Network as any)(
    containerRef.value,
    { nodes, edges } as any,
    DEFAULT_GRAPH_OPTIONS,
  )

  // 点击事件：编辑角色（连线模式下 canvas 已禁用指针事件，不会触发此处）
  network.on('click', (params: any) => {
    if (connectMode.value) return // 安全兜底，连线模式不处理
    if (params.nodes.length === 0) return
    const nodeIdx = params.nodes[0] as number
    // 普通模式：编辑角色
    openEdit(nodeIdx)
  })

  // improvedLayout 完成但物理还未启动时居中，确保首次出现就在可见区域中央
  network.once('startStabilization', () => {
    network?.fit({ animation: false })
  })

  // 物理模拟稳定后再次居中并关闭物理引擎，防止后续漂移
  network.once('stabilizationIterationsDone', () => {
    network?.fit({ animation: true })
    network?.setOptions({ physics: { enabled: false } })
    // 如果重建前处于连线模式，重建后自动恢复（实现连续建关系）
    if (wasConnect) nextTick(enterConnectMode)
  })

  // 同步 SVG viewBox 到容器尺寸，确保坐标系与像素匹配
  nextTick(updateSvgViewBox)
}

// 重建网络（加防抖避免频繁重建）
let rebuildTimer: ReturnType<typeof setTimeout> | null = null
function scheduleRebuild() {
  graphBuilt = false
  if (rebuildTimer) clearTimeout(rebuildTimer)
  rebuildTimer = setTimeout(() => {
    if (containerRef.value) ensureGraphBuilt()
  }, 100)
}

watch(() => props.characters, () => {
  nextTick(scheduleRebuild)
}, { deep: true })

// 关系变化时重建图谱，确保添加连线后立即显示
watch(() => props.relationships, () => {
  nextTick(scheduleRebuild)
}, { deep: true })

onMounted(() => {
  if (!containerRef.value) return
  // 容器在挂载时可能已有尺寸，立即尝试构建（比等 ResizeObserver 回调更快）
  if (!graphBuilt) {
    const rect = containerRef.value.getBoundingClientRect()
    if (rect.width > 0 && rect.height > 0) ensureGraphBuilt()
  }
  // 同时用 ResizeObserver 兜底：容器还未渲染完毕就等它出现尺寸后再构建
  // 构建后继续监听容器高度变化重新居中
  resizeObserver = new ResizeObserver((entries) => {
    for (const entry of entries) {
      if (entry.contentRect.width > 0 && entry.contentRect.height > 0) {
        if (!graphBuilt) {
          ensureGraphBuilt()
        } else {
          scheduleResizeFit()
        }
      }
    }
  })
  resizeObserver.observe(containerRef.value)
})

onUnmounted(() => {
  if (network) network.destroy()
  if (resizeObserver) resizeObserver.disconnect()
  if (rebuildTimer) clearTimeout(rebuildTimer)
  if (resizeFitTimer) clearTimeout(resizeFitTimer)
  removeDragListeners()
})

/** 重新布局（用户手动触发） */
function reLayout() {
  if (network) {
    network.setOptions({ physics: { enabled: true } })
    network.once('stabilizationIterationsDone', () => {
      network?.fit({ animation: true })
      network?.setOptions({ physics: { enabled: false } })
      updateSvgViewBox()
    })
  }
}
</script>

<template>
  <div class="graph-wrapper">
    <!-- 顶部工具栏 -->
    <div class="graph-toolbar">
      <n-button size="small" type="primary" @click="openAdd">
        <template #icon><n-icon><AddIcon/></n-icon></template>添加角色
      </n-button>
      <n-button size="small" secondary @click="reLayout" :disabled="!network">
        重新布局
      </n-button>
      <n-button size="small" :type="connectMode ? 'warning' : 'default' as any" @click="handleConnectClick" :disabled="characters.length === 0">
        连线
      </n-button>
      <n-text v-if="characters.length > 0" depth="3" style="font-size: 13px;">
        {{ characters.length }} 个角色 · 点击节点编辑 · 拖拽连线建关系
      </n-text>
    </div>

    <!-- vis-network 画布 + SVG 拖拽叠加层（SVG 在外层避免 vis-network DOM 干扰） -->
    <div v-if="characters.length > 0" class="graph-area">
      <div ref="containerRef" class="graph-container" :class="{ 'connect-mode': connectMode }" />
      <svg class="graph-svg-overlay" :viewBox="svgViewBox">
        <defs>
          <marker id="drag-arrow" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
            <polygon points="0 0, 10 3.5, 0 7" fill="#2080f0" />
          </marker>
        </defs>
        <!-- 拖拽预览线：内联属性避免 scoped CSS 不作用到 SVG 元素 -->
        <line v-if="dragActive"
          :x1="dragLine.x1" :y1="dragLine.y1"
          :x2="dragLine.x2" :y2="dragLine.y2"
          stroke="#2080f0" stroke-width="2.5"
          stroke-dasharray="6, 4" stroke-linecap="round"
          marker-end="url(#drag-arrow)" />
      </svg>
    </div>
    <!-- 有角色但无关系时显示提示（置于图下方，不遮挡） -->
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

        <!-- 关系子列表 -->
        <n-form-item label="与其他角色的关系">
          <div class="relationship-list">
            <div
              v-for="(rel, ri) in editRelationships"
              :key="ri"
              class="relationship-row"
            >
              <n-select
                v-model:value="rel.targetName"
                :options="otherCharNames.map(n => ({ label: n, value: n }))"
                placeholder="选择角色"
                filterable
                style="width: 130px;"
                size="small"
              />
              <n-button text size="small" style="width: 28px; font-size: 16px; flex-shrink: 0;" @click="rel.isIncoming = !rel.isIncoming" :title="rel.isIncoming ? '入向（←）' : '出向（→）'">
                {{ rel.isIncoming ? '←' : '→' }}
              </n-button>
              <n-input
                v-model:value="rel.relationType"
                placeholder="关系类型"
                size="small"
                style="width: 100px;"
              />
              <n-input
                v-model:value="rel.description"
                placeholder="描述"
                size="small"
                style="flex: 1; min-width: 0;"
              />
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

<style scoped>
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
}
.graph-container.connect-mode {
  cursor: crosshair;
}

/* SVG 拖拽线叠加层：仅用于视觉绘制，不拦截鼠标事件 */
.graph-svg-overlay {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 10;
}
/* 无关系提示 — 置于图容器下方，不遮挡画布 */
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
