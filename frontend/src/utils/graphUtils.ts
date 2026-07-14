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

/** 估算节点尺寸（基于中文字符宽度） */
function estimateNodeSize(name: string): { width: number; height: number } {
  const charWidth = name.length * 15 + 28 // 字符宽度 + 内边距
  return {width: Math.max(120, Math.min(charWidth, 220)), height: 44}
}

/** 构建 X6 `rect` 节点配置数组 */
export function buildNodes(chars: Character[]) {
  return chars.map((ch, i) => {
    const c = nodeColor(ch.name)
    const {width, height} = estimateNodeSize(ch.name)
    // 初始位置暂放 (0,0)，构建完成后由 applyCircleLayout 重新排列
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
        index: i,
        name: ch.name,
        alias: ch.alias,
        traits: ch.traits,
        description: ch.description,
      } satisfies X6NodeMeta,
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

/** 构建 X6 `edge` 配置数组 */
export function buildEdges(chars: Character[], rels?: NovelRelationship[]) {
  const edges: any[] = []
  if (!rels) return edges
  rels.forEach((rel, i) => {
    const sourceIdx = chars.findIndex(c => c.name === rel.source)
    const targetIdx = chars.findIndex(c => c.name === rel.target)
    if (sourceIdx < 0 || targetIdx < 0) return
    edges.push({
      id: `edge-${i}`,
      shape: 'edge' as const,
      zIndex: 1, // 低于节点的 99，确保节点和端口在边之上
      source: `char-${sourceIdx}`,
      target: `char-${targetIdx}`,
      attrs: {
        line: {
          stroke: '#888',
          strokeWidth: 2,
          strokeDasharray: undefined,
          targetMarker: null, // 无向连线，去掉箭头
        },
      },
      connector: {name: 'smooth'}, // 曲线连线
      labels: [
        {
          attrs: {
            text: {
              text: rel.relationType,
              fill: '#666',
              fontSize: 12,
              fontFamily: 'sans-serif',
            },
            rect: {
              fill: '#fff',
              stroke: '#ddd',
              rx: 4,
              ry: 4,
            },
          },
          position: 0.5,
        },
      ],
      data: {sourceIdx, targetIdx},
    })
  })
  return edges
}
