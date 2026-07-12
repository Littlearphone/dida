<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import {
  NButton, NIcon, NModal, NForm, NFormItem, NInput, NSelect,
  NSpace, useMessage, NEmpty, NText, NPopconfirm,
} from 'naive-ui'
import {
  AddOutline as AddIcon, TrashOutline as DeleteIcon,
} from '@vicons/ionicons5'
import { Network } from 'vis-network'
import { DataSet } from 'vis-data'
import type { Character, Relationship } from '../../types'


/** vis-network 节点数据结构 */
interface VisNodeItem {
  id: number
  label: string
  title?: string
  shape: string
  size: number
  font: { size: number; face: string }
  borderWidth: number
  color: { background: string; border: string; highlight: { background: string; border: string } }
}
/** vis-network 边数据结构 */
interface VisEdgeItem {
  from: number
  to: number
  label?: string
  title?: string
  font: { size: number; align: string }
  smooth: { type: string; roundness: number }
  color: { color: string; highlight: string }
  width: number
}

const props = defineProps<{ characters: Character[] }>()
const emit = defineEmits<{
  'update:characters': [chars: Character[]]
}>()

const message = useMessage()
const containerRef = ref<HTMLDivElement>()
let network: Network | null = null

// ------ 角色编辑弹框 ------
const showEdit = ref(false)
const editingIndex = ref(-1)
const editName = ref('')
const editAlias = ref('')
const editTraits = ref('')
const editDesc = ref('')
const editRelationships = ref<Relationship[]>([])

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
}

// ------ vis-network ------
function buildGraph() {
  if (!containerRef.value) return

  // 清理旧实例
  if (network) {
    network.destroy()
    network = null
  }

  const chars = props.characters
  if (chars.length === 0) return

  const nodeItems: VisNodeItem[] = chars.map((ch, i) => ({
    id: i,
    label: ch.name,
    title: [ch.alias, ch.traits].filter(Boolean).join('\n'),
    shape: 'ellipse',
    size: 25,
    font: { size: 14, face: 'sans-serif' },
    borderWidth: 2,
    color: {
      background: '#d4e8ff',
      border: '#2080f0',
      highlight: { background: '#b3d6ff', border: '#2080f0' },
    },
  }))

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
          font: { size: 12, align: 'middle' },
          smooth: { type: 'curvedCW', roundness: 0.15 },
          color: { color: '#999', highlight: '#2080f0' },
          width: 1.5,
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
        solver: 'forceAtlas2Based',
        forceAtlas2Based: { gravitationalConstant: -40, springLength: 180, springConstant: 0.02 },
        stabilization: { iterations: 200 },
      },
      layout: { improvedLayout: false },
      interaction: {
        hover: true,
        tooltipDelay: 200,
        zoomView: true,
        dragView: true,
      },
      edges: { arrows: { to: { enabled: true, scaleFactor: 0.6 } } },
    },
  )

  // 点击节点 → 编辑角色
  network.on('click', (params) => {
    if (params.nodes.length > 0) {
      openEdit(params.nodes[0] as number)
    }
  })
}

// 重建网络（加防抖避免频繁重建）
let rebuildTimer: ReturnType<typeof setTimeout> | null = null
function scheduleRebuild() {
  if (rebuildTimer) clearTimeout(rebuildTimer)
  rebuildTimer = setTimeout(() => {
    if (containerRef.value) buildGraph()
  }, 100)
}

watch(() => props.characters, () => {
  nextTick(scheduleRebuild)
}, { deep: true })

onMounted(() => {
  buildGraph()
})

onUnmounted(() => {
  if (network) network.destroy()
  if (rebuildTimer) clearTimeout(rebuildTimer)
})

/** 重新布局（用户手动触发） */
function reLayout() {
  if (network) {
    network.setOptions({ physics: { enabled: true } })
    setTimeout(() => network?.setOptions({ physics: { enabled: false } }), 1000)
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
      <n-text v-if="characters.length > 0" depth="3" style="font-size: 13px;">
        {{ characters.length }} 个角色 · 点击节点编辑
      </n-text>
    </div>

    <!-- vis-network 画布 -->
    <div v-if="characters.length > 0" ref="containerRef" class="graph-container" />
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

/* vis-network 画布填满剩余空间 */
.graph-container {
  flex: 1;
  min-height: 0;
  border: 1px solid #eee;
  border-radius: 6px;
  background: #fafafa;
  overflow: hidden;
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
