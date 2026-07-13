/**
 * 角色编辑弹框 composable
 * 管理：添加/编辑角色弹框状态、角色 CRUD、关系子列表
 */
import { ref } from 'vue'
import type { Character, NovelRelationship } from '../types'

interface EditRel {
  targetName: string
  relationType: string
  description?: string
  isIncoming?: boolean
}

export function useCharacterEdit(
  characters: { value: Character[] },
  relationships: { value: NovelRelationship[] | undefined },
  emit: any,
  message: any,
  autoSave: (chars: Character[], rels?: NovelRelationship[]) => void,
) {
  const showEdit = ref(false)
  const editingIndex = ref(-1)
  const editName = ref('')
  const editAlias = ref('')
  const editTraits = ref('')
  const editDesc = ref('')
  const editRelationships = ref<EditRel[]>([])
  const otherCharNames = ref<string[]>([])

  function openAdd() {
    editingIndex.value = -1
    editName.value = ''
    editAlias.value = ''
    editTraits.value = ''
    editDesc.value = ''
    editRelationships.value = []
    otherCharNames.value = characters.value.map(c => c.name)
    showEdit.value = true
  }

  function openEdit(index: number) {
    const ch = characters.value[index]
    editingIndex.value = index
    editName.value = ch.name
    editAlias.value = ch.alias || ''
    editTraits.value = ch.traits || ''
    editDesc.value = ch.description || ''
    // 显示所有涉及该角色的关系（出向 + 入向），isIncoming 标记方向
    editRelationships.value = (relationships.value || [])
      .filter(r => r.source === ch.name || r.target === ch.name)
      .map(r => ({
        targetName: r.source === ch.name ? r.target : r.source,
        relationType: r.relationType,
        description: r.description,
        isIncoming: r.target === ch.name,
      }))
    // 排除自身
    otherCharNames.value = characters.value
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
    const dup = characters.value.findIndex(
      (c, i) => c.name === editName.value.trim() && i !== editingIndex.value,
    )
    if (dup >= 0) {
      message.warning('角色名称已存在')
      return
    }
    const charName = editName.value.trim()
    const list = [...characters.value]
    // 对别名/特征/描述做逐项去重，清理历史遗留的重复内容
    const dedupParts = (val: string, sep: RegExp) =>
      [...new Set(val.split(sep).map(s => s.trim()).filter(Boolean))].join(sep.source === '\\n' ? '\n' : '、')
    const ch: Character = {
      name: charName,
      alias: editAlias.value.trim() ? dedupParts(editAlias.value.trim(), /[、，,]/) : undefined,
      traits: editTraits.value.trim() ? dedupParts(editTraits.value.trim(), /[、，,]/) : undefined,
      description: editDesc.value.trim() ? dedupParts(editDesc.value.trim(), /\n/) : undefined,
    }
    if (editingIndex.value >= 0) {
      list[editingIndex.value] = ch
    } else {
      list.push(ch)
    }

    // 移除此角色相关的所有旧关系（含作为 source 和 target），再按编辑后的方向写回
    const oldName = editingIndex.value >= 0 ? characters.value[editingIndex.value].name : ''
    let newRels = (relationships.value || []).filter(r => r.source !== oldName && r.target !== oldName)
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
    const ch = characters.value[index]
    // 检查是否有涉及该角色的关系
    const involvedRels = (relationships.value || []).filter(r => r.source === ch.name || r.target === ch.name)
    if (involvedRels.length > 0) {
      if (!window.confirm(`角色「${ch.name}」存在 ${involvedRels.length} 条关系记录。删除后这些关系将被清理，是否继续？`)) {
        return
      }
    }
    const list = [...characters.value]
    list.splice(index, 1)
    // 清理涉及该角色的所有关系
    const newRels = (relationships.value || []).filter(r => r.source !== ch.name && r.target !== ch.name)

    emit('update:characters', list)
    emit('update:relationships', newRels)
    autoSave(list, newRels)
  }

  return {
    showEdit, editingIndex,
    editName, editAlias, editTraits, editDesc,
    editRelationships, otherCharNames,
    openAdd, openEdit,
    addRelationship, removeRelationship,
    saveCharacter, removeCharacter,
  }
}
