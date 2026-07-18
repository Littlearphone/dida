/**
 * AntV X6 角色关系图节点/边配置工厂
 */
import type {Character, NovelRelationship} from '../types'

/** 节点颜色方案 */
const COLOR_PALETTE = [
  {bg: '#e8f4f8', border: '#2980b9'},
  {bg: '#fce4ec', border: '#c0392b'},
  {bg: '#e8f5e9', border: '#27ae60'},
  {bg: '#f3e5f5', border: '#8e44ad'},
  {bg: '#fff3e0', border: '#d35400'},
  {bg: '#e0f7fa', border: '#16a085'},
  {bg: '#fffde7', border: '#d4ac0d'},
  {bg: '#eceff1', border: '#546e7a'},
]

/** 基于角色名生成确定的颜色 */
export function nodeColor(name: string) {
  const hash = name.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0)
  return COLOR_PALETTE[hash % COLOR_PALETTE.length]
}

/** 附加在 X6 节点 data 上的角色元信息 */
export interface X6NodeMeta {
  index: number
  name: string
  alias?: string
  traits?: string
  description?: string
}

/** 估算节点尺寸（基于中文字符宽度，加宽确保名字完整显示） */
function estimateNodeSize(name: string): { width: number; height: number } {
  const charWidth = name.length * 16 + 36 // 每字 16px + 36px 内边距
  return {width: Math.max(130, Math.min(charWidth, 300)), height: 46}
}

/** 构建 X6 `rect` 节点配置数组。origIdxFn 可选，用于子图场景下映射回原始角色索引 */
export function buildNodes(chars: Character[], origIdxFn?: (subIdx: number) => number) {
  return chars.map((ch, i) => {
    const c = nodeColor(ch.name)
    const {width, height} = estimateNodeSize(ch.name)
    return {
      id: `char-${i}`,
      shape: 'rect' as const,
      x: 0,
      y: 0,
      width,
      height,
      zIndex: 99, // 节点在边之上，端口不被边遮挡
      attrs: {
        body: {
          fill: c.bg,
          stroke: c.border,
          strokeWidth: 2,
          rx: 8,
          ry: 8,
        },
        label: {
          text: ch.name,
          fill: '#333',
          fontSize: 14,
          fontFamily: 'sans-serif',
          textAnchor: 'middle',
          textVerticalAnchor: 'middle',
        },
      },
      data: {
        index: origIdxFn ? origIdxFn(i) : i,
        name: ch.name,
        alias: ch.alias,
        traits: ch.traits,
        description: ch.description,
        color: c,
      } satisfies X6NodeMeta & { color: typeof c },
      // 四边连线端口（拖拽小圆点，hover 节点后可见）
      ports: {
        groups: {
          edge: {
            position: { name: 'absolute' },
            attrs: {
              circle: {
                r: 5,
                magnet: true, // 标记为可连线磁铁
                fill: '#2080f0',
                stroke: '#fff',
                strokeWidth: 2,
                cursor: 'crosshair',
              },
            },
          },
        },
        items: [
          { id: `${i}-t`, group: 'edge', args: { x: '50%', y: 0 } },     // 上边中点
          { id: `${i}-r`, group: 'edge', args: { x: '100%', y: '50%' } }, // 右边中点
          { id: `${i}-b`, group: 'edge', args: { x: '50%', y: '100%' } }, // 下边中点
          { id: `${i}-l`, group: 'edge', args: { x: 0, y: '50%' } },      // 左边中点
        ],
      },
    }
  })
}

/** 构建 X6 `edge` 配置数组（平行边标签自动偏移、连线由布局层负责外绕） */
export function buildEdges(chars: Character[], rels?: NovelRelationship[]) {
  const edges: any[] = []
  if (!rels) return edges

  // 统计同一对节点之间的边数，用于平行边标签偏移
  const pairCounts = new Map<string, number>()
  rels.forEach(rel => {
    const si = chars.findIndex(c => c.name === rel.source)
    const ti = chars.findIndex(c => c.name === rel.target)
    if (si < 0 || ti < 0) return
    const key = si < ti ? `${si}-${ti}` : `${ti}-${si}`
    pairCounts.set(key, (pairCounts.get(key) || 0) + 1)
  })
  const pairIndexes = new Map<string, number>()

  rels.forEach((rel, i) => {
    const sourceIdx = chars.findIndex(c => c.name === rel.source)
    const targetIdx = chars.findIndex(c => c.name === rel.target)
    if (sourceIdx < 0 || targetIdx < 0) return

    const pairKey = sourceIdx < targetIdx ? `${sourceIdx}-${targetIdx}` : `${targetIdx}-${sourceIdx}`
    const pairTotal = pairCounts.get(pairKey) || 1
    const pairSeq = pairIndexes.get(pairKey) || 0
    pairIndexes.set(pairKey, pairSeq + 1)

    // 平行边标签偏移：同一对角色多条关系时标签上下错开不重叠
    const offset = pairTotal > 1
      ? (pairSeq - (pairTotal - 1) / 2) * 24
      : 0

    edges.push({
      id: `edge-${i}`,
      shape: 'edge' as const,
      zIndex: 1,
      source: `char-${sourceIdx}`,
      target: `char-${targetIdx}`,
      attrs: {
        line: {
          stroke: '#bbb',
          strokeWidth: 1.2,
          strokeDasharray: undefined,
          targetMarker: null,
        },
      },
      connector: { name: 'smooth' },
      labels: [
        {
          attrs: {
            text: {
              text: rel.relationType,
              fill: '#999',
              fontSize: 11,
              fontFamily: 'sans-serif',
            },
            rect: {
              fill: '#fafafa',
              stroke: '#e0e0e0',
              rx: 3,
              ry: 3,
            },
          },
          position: { distance: 0.5, offset },
        },
      ],
      data: { sourceIdx, targetIdx },
    })
  })
  return edges
}

/** 基于关系邻接度对角色重排序：关系近的放一起，减少环内连线交叉 */
export function sortByAdjacency(chars: Character[], rels?: NovelRelationship[]): number[] {
  if (!rels || rels.length === 0 || chars.length <= 2) {
    return chars.map((_, i) => i)
  }

  // 构建邻接矩阵（无向，权重 = 关系数）
  const adj = new Map<string, number>()
  rels.forEach(r => {
    const si = chars.findIndex(c => c.name === r.source)
    const ti = chars.findIndex(c => c.name === r.target)
    if (si < 0 || ti < 0) return
    const key = si < ti ? `${si}-${ti}` : `${ti}-${si}`
    adj.set(key, (adj.get(key) || 0) + 1)
  })

  // 贪心排列：从关系最多的角色开始，每次选与已排角色邻接权重最高的
  const placed = new Set<number>()
  const order: number[] = []

  // 选邻接权重最大的节点作为起点
  let maxDeg = -1, start = 0
  for (let i = 0; i < chars.length; i++) {
    let deg = 0
    for (let j = 0; j < chars.length; j++) {
      if (i === j) continue
      const key = i < j ? `${i}-${j}` : `${j}-${i}`
      deg += adj.get(key) || 0
    }
    if (deg > maxDeg) { maxDeg = deg; start = i }
  }

  order.push(start)
  placed.add(start)

  while (placed.size < chars.length) {
    let bestIdx = -1, bestWeight = -1
    for (let i = 0; i < chars.length; i++) {
      if (placed.has(i)) continue
      // 计算与已放置节点的总邻接权重
      let weight = 0
      for (const p of placed) {
        const key = i < p ? `${i}-${p}` : `${p}-${i}`
        weight += adj.get(key) || 0
      }
      if (weight > bestWeight) { bestWeight = weight; bestIdx = i }
    }
    // 若无关系，取第一个未放置的
    if (bestIdx < 0) {
      for (let i = 0; i < chars.length; i++) {
        if (!placed.has(i)) { bestIdx = i; break }
      }
    }
    order.push(bestIdx)
    placed.add(bestIdx)
  }

  return order
}
