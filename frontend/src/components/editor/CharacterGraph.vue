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
import type { Character, Relationship } from '../../types'
import { useNovelStore } from '../../stores/novel'


/** vis-network 节点数据结构 */
interface VisNodeItem {
  id: number
  label: string
  title?: string
  shape: string
  size: number
  font: { size: number; face: string; color?: string; multi?: boolean }
  borderWidth: number
  borderRadius?: number
  margin?: { top: number; bottom: number; left: number; right: number }
  color: { background: string; border: string; highlight: { background: string; border: string } }
  shadow?: { enabled: boolean; color: string; size: number; x: number; y: number }
}
/** vis-network 边数据结构 */
interface VisEdgeItem {
  from: number
  to: number
  label?: string
  title?: string
  font: { size: number; align: string; color?: string }
  smooth: { type: string; roundness: number }
  color: { color: string; highlight: string }
  width: number
  arrows?: string
  arrowStrikethrough?: boolean
}

const props = defineProps<{
  characters: Character[]
  /** 提供 novelId 时，角色变更（新增/编辑/删除）会自动持久化到后端 */
  novelId?: string
}>()
const emit = defineEmits<{
  'update:characters': [chars: Character[]]
}>()

const message = useMessage()
const novelStore = useNovelStore()
const containerRef = ref<HTMLDivElement>()
let network: Network | null = null

/** 是否有已定义的关系 */
const hasRelationships = computed(() =>
  props.characters.some(c => c.relationships && c.relationships.length > 0),
)

// ------ 角色编辑弹框 ------
const showEdit = ref(false)
const editingIndex = ref(-1)
const editName = ref('')
const editAlias = ref('')
const editTraits = ref('')
const editDesc = ref('')
const editRelationships = ref<Relationship[]>([])

// ------ 连线模式（直接在图上点节点建关系）------
const connectMode = ref(false)
const connectSrcIdx = ref(-1)
const connectTgtIdx = ref(-1)
const showConnectDialog = ref(false)
const connectType = ref('')
const connectDesc = ref('')

function toggleConnectMode() {
  connectMode.value = !connectMode.value
  if (connectMode.value) {
    connectSrcIdx.value = -1
    message.info('连线模式：点击一个角色作为关系起点')
  }
}

function cancelConnect() {
  connectMode.value = false
  showConnectDialog.value = false
  connectSrcIdx.value = -1
  connectTgtIdx.value = -1
  connectType.value = ''
  connectDesc.value = ''
}

function confirmConnect() {
  if (!connectType.value.trim()) {
    message.warning('请输入关系类型')
    return
  }
  const src = connectSrcIdx.value
  const tgt = connectTgtIdx.value
  const list = [...props.characters]
  if (!list[src].relationships) list[src].relationships = []
  list[src].relationships.push({
    targetName: props.characters[tgt].name,
    relationType: connectType.value.trim(),
    description: connectDesc.value.trim() || undefined,
  })
  emit('update:characters', list)
  autoSave(list)
  message.success(`已添加「${props.characters[src].name}」→「${props.characters[tgt].name}」`)
  cancelConnect()
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
  editRelationships.value = JSON.parse(JSON.stringify(ch.relationships || []))
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
  const list = [...props.characters]
  const ch: Character = {
    name: editName.value.trim(),
    alias: editAlias.value.trim() || undefined,
    traits: editTraits.value.trim() || undefined,
    description: editDesc.value.trim() || undefined,
    relationships: editRelationships.value.filter(r => r.targetName.trim()) || undefined,
  }
  if (editingIndex.value >= 0) {
    list[editingIndex.value] = ch
  } else {
    list.push(ch)
  }
  emit('update:characters', list)
  showEdit.value = false
  // 自动保存到后端（提供 novelId 时）
  autoSave(list)
}

function removeCharacter(index: number) {
  const ch = props.characters[index]
  // 检查是否有其他角色的关系引用该角色
  const refs = props.characters
    .map((c, i) => ({ name: c.name, i }))
    .filter(({ name }) =>
      name !== ch.name && props.characters.some(
        c2 => c2.relationships?.some(r => r.targetName === ch.name),
      ),
    )
  if (refs.length > 0) {
    const names = refs.map(r => r.name).join('、')
    if (!window.confirm(`角色「${ch.name}」被其他角色（${names}）的关系引用。删除后将自动清理这些引用，是否继续？`)) {
      return
    }
  }
  const list = [...props.characters]
  list.splice(index, 1)
  // 清理其他角色中对该角色的关系引用
  const targetName = ch.name
  for (const c of list) {
    if (c.relationships) {
      c.relationships = c.relationships.filter(r => r.targetName !== targetName)
      if (c.relationships.length === 0) c.relationships = undefined
    }
  }
  emit('update:characters', list)
  // 自动保存到后端（提供 novelId 时）
  autoSave(list)
}

/** 有 novelId 时自动将角色数据持久化到后端 */
async function autoSave(chars: Character[]) {
  if (!props.novelId) return
  const ok = await novelStore.updateNovelMeta(props.novelId, { characters: chars })
  if (!ok) message.error('角色数据保存失败')
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
  }, 150)
}

/** 基于名称生成节点颜色 */
function nodeColor(name: string) {
  const palette = [
    { bg: '#e8f4f8', border: '#2980b9' },
    { bg: '#fce4ec', border: '#c0392b' },
    { bg: '#e8f5e9', border: '#27ae60' },
    { bg: '#f3e5f5', border: '#8e44ad' },
    { bg: '#fff3e0', border: '#d35400' },
    { bg: '#e0f7fa', border: '#16a085' },
    { bg: '#fffde7', border: '#d4ac0d' },
    { bg: '#eceff1', border: '#546e7a' },
  ]
  const hash = name.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0)
  return palette[hash % palette.length]
}

function buildGraph() {
  if (!containerRef.value) return

  // 重建时退出连线模式
  cancelConnect()

  // 清理旧实例
  if (network) {
    network.destroy()
    network = null
  }

  const chars = props.characters
  if (chars.length === 0) return

  const nodeItems: VisNodeItem[] = chars.map((ch, i) => {
    const c = nodeColor(ch.name)
    return {
      id: i,
      label: ch.name,
      title: [ch.alias, `「${ch.traits}」`, ch.description].filter(Boolean).join('\n'),
      shape: 'box',
      size: 30,
      font: { size: 14, face: 'sans-serif', color: '#333', multi: false },
      borderWidth: 2,
      borderRadius: 8,
      margin: { top: 8, bottom: 8, left: 12, right: 12 },
      color: { background: c.bg, border: c.border, highlight: { background: c.bg, border: c.border } },
      shadow: { enabled: true, color: 'rgba(0,0,0,0.1)', size: 4, x: 0, y: 2 },
    }
  })

  const edgeItems: VisEdgeItem[] = []
  chars.forEach((ch, i) => {
    if (!ch.relationships) return
    ch.relationships.forEach(rel => {
      const targetIdx = chars.findIndex(c => c.name === rel.targetName)
      if (targetIdx >= 0) {
        edgeItems.push({
          from: i,
          to: targetIdx,
          label: rel.relationType,
          title: rel.description || undefined,
          font: { size: 12, align: 'middle', color: '#666' },
          smooth: { type: 'curvedCW', roundness: 0.12 },
          color: { color: '#888', highlight: '#2080f0' },
          width: 2,
          arrows: 'to',
          arrowStrikethrough: false,
        })
      }
    })
  })

  // 用 as any 绕过 vis-data 严格类型（边没有 id 字段，仅有 from/to）
  const nodes = new DataSet(nodeItems as any)
  const edges = new DataSet(edgeItems as any)

  network = new Network(
    containerRef.value,
    // 使用类型断言绕过 vis 类型推断限制
    { nodes, edges } as any,
    {
      physics: {
        enabled: true,
        solver: 'barnesHut',
        barnesHut: { gravitationalConstant: -2000, centralGravity: 0.3, springLength: 160, springConstant: 0.02 },
        stabilization: { iterations: 300 },
      },
      layout: { improvedLayout: true },
      interaction: {
        hover: true,
        tooltipDelay: 200,
        zoomView: true,
        dragView: true,
      },
      edges: { smooth: true },
    },
  )

  // 点击事件：普通模式编辑角色，连线模式建关系
  network.on('click', (params) => {
    if (params.nodes.length === 0) {
      // 点击空白 — 如果处于连线模式则退出
      if (connectMode.value) cancelConnect()
      return
    }
    const nodeIdx = params.nodes[0] as number

    if (connectMode.value) {
      // 连线模式
      if (connectSrcIdx.value === -1) {
        // 第一步：选择起点
        connectSrcIdx.value = nodeIdx
        message.info(`已选择「${props.characters[nodeIdx].name}」，请点击关系目标`)
      } else if (connectSrcIdx.value === nodeIdx) {
        // 点了同一个节点 → 取消选择
        connectSrcIdx.value = -1
        message.info('已取消选择，重新点击起点')
      } else {
        // 第二步：选择终点 → 弹出关系类型输入
        connectTgtIdx.value = nodeIdx
        showConnectDialog.value = true
      }
      return
    }

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
  })
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

onMounted(() => {
  // 用 ResizeObserver 等容器有真实尺寸后再构建；构建后继续监听容器高度变化重新居中
  if (!containerRef.value) return
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
})

/** 重新布局（用户手动触发） */
function reLayout() {
  if (network) {
    network.setOptions({ physics: { enabled: true } })
    network.once('stabilizationIterationsDone', () => {
      network?.fit({ animation: true })
      network?.setOptions({ physics: { enabled: false } })
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
      <n-button size="small" :type="connectMode ? 'warning' : 'secondary'" @click="toggleConnectMode" :disabled="!network">
        连线
      </n-button>
      <n-text v-if="characters.length > 0" depth="3" style="font-size: 13px;">
        {{ characters.length }} 个角色 · 点击节点编辑 · 连线模式建关系
      </n-text>
    </div>

    <!-- vis-network 画布 -->
    <div v-if="characters.length > 0" class="graph-area">
      <div ref="containerRef" class="graph-container" :class="{ 'connect-mode': connectMode }" />
    </div>
    <!-- 有角色但无关系时显示提示（置于图下方，不遮挡） -->
    <div v-if="characters.length > 0 && !hasRelationships" class="graph-hint">
      角色已就绪，点击「连线」按钮在图上直接建立关系
    </div>
    <n-empty v-else description="还没有角色" class="graph-empty">
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
  border: 1px solid #eee;
  border-radius: 6px;
  background: #fafafa;
  overflow: hidden;
}
.graph-container.connect-mode {
  cursor: crosshair;
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
