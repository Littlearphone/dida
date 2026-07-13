/**
 * vis-network 角色关系图配置与节点数据生成工具
 */
import type { Character, NovelRelationship } from '../../types'

/** 节点颜色方案 */
const COLOR_PALETTE = [
  { bg: '#e8f4f8', border: '#2980b9' },
  { bg: '#fce4ec', border: '#c0392b' },
  { bg: '#e8f5e9', border: '#27ae60' },
  { bg: '#f3e5f5', border: '#8e44ad' },
  { bg: '#fff3e0', border: '#d35400' },
  { bg: '#e0f7fa', border: '#16a085' },
  { bg: '#fffde7', border: '#d4ac0d' },
  { bg: '#eceff1', border: '#546e7a' },
]

/** 基于角色名生成确定的颜色 */
export function nodeColor(name: string) {
  const hash = name.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0)
  return COLOR_PALETTE[hash % COLOR_PALETTE.length]
}

/** vis-network 节点项类型 */
export interface GraphNode {
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

/** vis-network 边项类型 */
export interface GraphEdge {
  from: number
  to: number
  label?: string
  title?: string
  font: { size: number; align: string; color?: string }
  smooth: { type: string; roundness: number }
  color: { color: string; highlight: string }
  width: number
}

/** 构建 vis-network 节点数据集 */
export function buildNodes(chars: Character[]): GraphNode[] {
  return chars.map((ch, i) => {
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
      color: {
        background: c.bg, border: c.border,
        highlight: { background: c.bg, border: c.border },
      },
      shadow: { enabled: true, color: 'rgba(0,0,0,0.1)', size: 4, x: 0, y: 2 },
    }
  })
}

/** 构建 vis-network 边数据集 */
export function buildEdges(chars: Character[], rels?: NovelRelationship[]): GraphEdge[] {
  const edges: GraphEdge[] = []
  if (!rels) return edges
  for (const rel of rels) {
    const sourceIdx = chars.findIndex(c => c.name === rel.source)
    const targetIdx = chars.findIndex(c => c.name === rel.target)
    if (sourceIdx < 0 || targetIdx < 0) continue
    edges.push({
      from: sourceIdx,
      to: targetIdx,
      label: rel.relationType,
      title: rel.description || undefined,
      font: { size: 12, align: 'middle', color: '#666' },
      smooth: { type: 'curvedCW', roundness: 0.12 },
      color: { color: '#888', highlight: '#2080f0' },
      width: 2,
    })
  }
  return edges
}

/** vis-network 默认选项 */
export const DEFAULT_GRAPH_OPTIONS = {
  physics: {
    enabled: true,
    solver: 'barnesHut' as const,
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
}
