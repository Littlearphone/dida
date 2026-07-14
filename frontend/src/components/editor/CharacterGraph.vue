<script setup lang="ts">
/**
 * 角色关系图谱 — 角色编辑 + AntV X6 可视化 + 连线建关系
 * 拖拽连线使用 X6 内置机制（connecting.allowNode），连线为曲线
 */
import { ref, computed, nextTick } from 'vue'
import {
  NButton, NIcon, NModal, NForm, NFormItem, NInput, NSelect,
  NSpace, useMessage, NEmpty, NText,
} from 'naive-ui'
import {
  AddOutline as AddIcon, GitMergeOutline as MergeIcon,
  RefreshOutline as ReloadIcon, TrashOutline as DeleteIcon,
} from '@vicons/ionicons5'
import type { Character, NovelRelationship } from '@/types'
import { useNovelStore } from '@/stores/novel.ts'
import { useCharacterEdit } from '@/composables/useCharacterEdit.ts'
import { useGraphNetwork } from '@/composables/useGraphNetwork.ts'

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
  saveCharacter,
} = useCharacterEdit(
  chars, rels, emit, message, autoSave,
)

// === 连线（X6 内置拖拽 → edge:connected → 弹框确认） ===
const connectSrcIdx = ref(-1)
const connectTgtIdx = ref(-1)
const connectEditIdx = ref(-1) // 修改已有关系时的索引，-1 表示新建
const showConnectDialog = ref(false)
const connectType = ref('')
const connectDesc = ref('')

/** 弹框标题：新建或修改 */
const connectDialogTitle = computed(() =>
  connectEditIdx.value >= 0 ? '修改关系' : '添加关系',
)

function resetConnectState() {
  showConnectDialog.value = false
  connectSrcIdx.value = -1
  connectTgtIdx.value = -1
  connectEditIdx.value = -1
  connectType.value = ''
  connectDesc.value = ''
}

function confirmConnect() {
  if (!connectType.value.trim()) {
    message.warning('请输入关系类型')
    return
  }
  const srcName = props.characters[connectSrcIdx.value].name
  const tgtName = props.characters[connectTgtIdx.value].name
  const rel: NovelRelationship = {
    source: srcName,
    target: tgtName,
    relationType: connectType.value.trim(),
    description: connectDesc.value.trim() || undefined,
  }

  let newRels: NovelRelationship[]
  if (connectEditIdx.value >= 0) {
    // 修改已有关系
    newRels = [...(props.relationships || [])]
    newRels[connectEditIdx.value] = rel
    message.success(`已修改「${srcName}」→「${tgtName}」`)
  } else {
    // 新建关系
    newRels = [...(props.relationships || []), rel]
    message.success(`已添加「${srcName}」→「${tgtName}」`)
  }
  emit('update:relationships', newRels)
  autoSave(props.characters, newRels)
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
/** 应用环状排列（手动计算位置，无物理引擎） */
function applyCircleLayout() {
  const graph = getGraph()
  if (!graph) return
  const el = containerRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const count = props.characters.length
  if (count < 2) return
  const cx = rect.width / 2
  const cy = rect.height / 2
  const radius = Math.min(cx, cy) - 80
  // 容器尺寸未稳定时跳过，等待下一次 resize 回调
  if (radius < 100) return
  props.characters.forEach((_, i) => {
    const angle = (2 * Math.PI * i) / count - Math.PI / 2
    const node = graph.getCellById(`char-${i}`)
    if (node && node.isNode()) {
      node.setPosition({ x: cx + radius * Math.cos(angle), y: cy + radius * Math.sin(angle) })
    }
  })
  graph.zoomToFit({ maxScale: 1, padding: 40 })
  graph.centerContent()
  // 将 fit 后的缩放设为最小缩放，防止缩过头（保留 5% 余量）
  const zoom = graph.zoom()
  if (zoom > 0) {
    ;(graph as any).options.scaling.min = zoom * 0.95
  }
}

// === 图谱 ===
const {
  graphReady,
  getGraph,
  setOnNodeClick,
  setOnEdgeConnected,
  setOnAfterBuild,
  setOnContainerResize,
} = useGraphNetwork(containerRef, chars, rels)

// 注册节点点击回调（打开编辑弹框）
setOnNodeClick((idx: number) => openEdit(idx))

// 接管 X6 内置拖拽连线结果：移除自动边 → 弹出关系编辑框
setOnEdgeConnected((edge: any) => {
  // 先读取元数据，再移除边（remove 后 getSourceCell 返回 null）
  const srcMeta = edge.getSourceCell()?.getData()
  const tgtMeta = edge.getTargetCell()?.getData()
  edge.remove()
  if (srcMeta === undefined || tgtMeta === undefined) return
  connectSrcIdx.value = srcMeta.index
  connectTgtIdx.value = tgtMeta.index

  // 检查是否有已存在的关系，有则进入修改模式
  const existingIdx = (props.relationships || []).findIndex(r =>
    (r.source === srcMeta.name && r.target === tgtMeta.name) ||
    (r.source === tgtMeta.name && r.target === srcMeta.name),
  )
  if (existingIdx >= 0) {
    connectEditIdx.value = existingIdx
    connectType.value = props.relationships![existingIdx].relationType
    connectDesc.value = props.relationships![existingIdx].description || ''
  } else {
    connectEditIdx.value = -1
    connectType.value = ''
    connectDesc.value = ''
  }

  showConnectDialog.value = true
})

// 图重建后自动恢复环状布局
setOnAfterBuild(() => { nextTick(applyCircleLayout) })

// 容器尺寸变化（如 Tab 切换动画）后重新布局
setOnContainerResize(() => { nextTick(applyCircleLayout) })
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
          @click="openMergeDialog"
          :disabled="characters.length < 2">
          <template #icon><n-icon><MergeIcon/></n-icon></template>合并
        </n-button>
        <n-button class="toolbar-btn" size="small" secondary
          @click="applyCircleLayout"
          :disabled="!graphReady">
          <template #icon><n-icon><ReloadIcon/></n-icon></template>重新布局
        </n-button>
      </div>
      <div class="toolbar-info">
        <n-text v-if="characters.length > 0" depth="3" style="font-size: 13px;">
          {{ characters.length }} 个角色
        </n-text>
      </div>
    </div>

    <!-- X6 画布（自带拖拽连线预览） -->
    <div v-if="characters.length > 0" class="graph-area">
      <div ref="containerRef" class="graph-container" />
    </div>
    <div v-if="characters.length > 0 && !hasRelationships" class="graph-hint">
      从节点拖拽到目标角色即可建立关系
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
    <n-modal :show="showConnectDialog" :title="connectDialogTitle" preset="card"
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
          <n-button quaternary @click="resetConnectState">取消</n-button>
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

/* X6 画布容器 */
.graph-area {
  flex: 1;
  min-height: 0;
  position: relative;
  display: flex;
  flex-direction: column;
}
.graph-container {
  flex: 1;
  min-height: 150px;
  position: relative;
  border: 1px solid #eee;
  border-radius: 6px;
  background: #fafafa;
  overflow: hidden;
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

<!-- 非 scoped：X6 端口 + 连线拖拽小圆点（SVG 内，不受 scoped 限制） -->
<style lang="scss">
/* 端口小圆点：hover 节点时显示，可拖拽连线 */
.x6-node {
  .x6-port-body {
    cursor: crosshair;
    width: 10px !important;
    height: 10px !important;
    margin-left: -5px !important;
    margin-top: -5px !important;
    border-radius: 50%;
    background: #2080f0;
    border: 2px solid #fff;
    box-shadow: 0 1px 3px rgba(0,0,0,.25);
    transition: opacity .15s, transform .15s, box-shadow .15s;
    opacity: 0;                       /* 默认隐藏，hover 节点时显示 */
  }
  &:hover .x6-port-body {
    opacity: 1;
  }
  .x6-port-body:hover {
    transform: scale(1.4);
    box-shadow: 0 2px 8px rgba(32,128,240,.5);
  }
}
</style>
