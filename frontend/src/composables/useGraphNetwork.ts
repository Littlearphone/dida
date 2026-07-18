/**
 * AntV X6 图谱生命周期 composable
 * 管理：Graph 构建、重建、ResizeObserver、重新布局
 * 连线使用 X6 内置拖拽机制，无需手动 mousedown 处理
 */
import {nextTick, onMounted, onUnmounted, ref, watch} from 'vue'
import {Graph, Shape} from '@antv/x6'
import type {X6NodeMeta} from '../utils/graphUtils'
import {buildEdges, buildNodes} from '../utils/graphUtils'
import type {Character, NovelRelationship} from '@/types'

export function useGraphNetwork(
  containerRef: { value: HTMLDivElement | undefined },
  characters: { value: Character[] },
  relationships: { value: NovelRelationship[] | undefined },
  origIdxFn?: (subIdx: number) => number,
) {
  let graph: Graph | null = null
  let graphBuilt = false
  let resizeObserver: ResizeObserver | null = null
  let rebuildTimer: ReturnType<typeof setTimeout> | null = null
  let resizeRelayoutTimer: ReturnType<typeof setTimeout> | null = null
  /** 响应式标志：Graph 是否已构建完成（使模板可响应式判断） */
  const graphReady = ref(false)

  function buildGraph() {
    if (!containerRef.value) return

    if (graph) { graph.dispose(); graph = null }

    const chars = characters.value
    if (chars.length === 0) return

    // 获取容器尺寸作为初始宽高
    const rect = containerRef.value.getBoundingClientRect()
    const width = Math.max(1, Math.round(rect.width))
    const height = Math.max(1, Math.round(rect.height))

    graph = new Graph({
      container: containerRef.value,
      width,
      height,
      autoResize: true,
      panning: true,
      mousewheel: { enabled: true, zoomAtMousePosition: true },
      scaling: { min: 0.95, max: 2.5 },
      grid: { visible: true },
      background: { color: '#fafafa' },
      // X6 内置拖拽连线：从端口拖到目标节点自动创建边
      connecting: {
        allowBlank: false,
        snap: true, // 连接时吸附到最近的端口
        highlight: true, // 悬停目标端口时高亮
        validateMagnet({ magnet }) {
          return true;
        },
        // 阻止自连，允许已存在关系的角色重新连线以触发修改
        validateConnection({ sourceCell, targetCell }) {
          if (!sourceCell || !targetCell) return false
          if (sourceCell.id === targetCell.id) return false
          return true
        },
        // 拖拽过程中的临时边样式（虚线预览）
        createEdge: () => new Shape.Edge({
          attrs: {
            line: {
              stroke: '#888',
              strokeWidth: 2,
              strokeDasharray: '5 5',
              targetMarker: null,
            },
          },
          connector: { name: 'smooth' },
        }),
      },
    })

    // 添加节点
    const nodeConfigs = buildNodes(chars, origIdxFn)
    nodeConfigs.forEach((cfg: any) => graph!.addNode(cfg))

    // 用户拖拽连线完成 → 弹出关系编辑对话框
    // edge:connected 只由拖拽触发（不是编程添加），isNew 标记新创建的边
    graph.on('edge:connected', ({ edge, isNew }: any) => {
      if (isNew) {
        onEdgeConnected(edge)
      }
    })

    // 添加边
    const edgeConfigs = buildEdges(chars, relationships.value)
    edgeConfigs.forEach((cfg: any) => graph!.addEdge(cfg))

    // 节点双击 → 编辑角色（传名字，调用方按名反查原始索引）
    graph.on('node:dblclick', ({ node }) => {
      const meta = node.getData()
      if (meta?.name != null) onNodeClick(meta.name)
    })

    // 节点单击 → 高亮关系
    graph.on('node:click', ({ node }: any) => {
      const meta = node.getData()
      if (meta?.name != null && onNodeSingleClick) {
        onNodeSingleClick(meta.name)
      }
    })

    // 点击画布空白 → 取消高亮
    graph.on('blank:click', () => {
      onBlankClick?.()
    })

    // 标记构建完成
    graphReady.value = true

    // 首次适应内容
    nextTick(() => {
      graph?.zoomToFit({ maxScale: 1, padding: 40 })
      graph?.centerContent()
    })
    nextTick(() => onAfterBuild?.())
  }

  let onNodeClick = (_nodeName: string) => {}
  let onNodeSingleClick: ((name: string) => void) | null = null
  let onBlankClick: (() => void) | null = null

  function setOnNodeClick(cb: (name: string) => void) {
    onNodeClick = cb
  }

  function setOnNodeSingleClick(cb: (name: string) => void) {
    onNodeSingleClick = cb
  }
  function setOnBlankClick(cb: () => void) {
    onBlankClick = cb
  }

  /** 拖拽连线完成回调（X6 自动创建边后触发），由使用者接管弹框流程 */
  let onEdgeConnected: (edge: any) => void = () => {}

  function setOnEdgeConnected(cb: (edge: any) => void) {
    onEdgeConnected = cb
  }

  /** 构建完成回调（用于恢复布局等操作） */
  let onAfterBuild: (() => void) | null = null

  function setOnAfterBuild(cb: () => void) {
    onAfterBuild = cb
  }

  /** 容器 resize 回调（用于 Tab 切换动画过渡后重新布局） */
  let onContainerResize: (() => void) | null = null

  function setOnContainerResize(cb: () => void) {
    onContainerResize = cb
  }

  function ensureGraphBuilt() {
    if (graphBuilt || !containerRef.value) return
    const rect = containerRef.value.getBoundingClientRect()
    if (rect.width === 0 || rect.height === 0) return
    graphBuilt = true
    buildGraph()
  }

  function scheduleRebuild() {
    graphBuilt = false
    graphReady.value = false  // 重建期间禁用按钮
    if (rebuildTimer) clearTimeout(rebuildTimer)
    rebuildTimer = setTimeout(() => {
      if (containerRef.value) ensureGraphBuilt()
    }, 100)
  }

  /** 获取 X6 Graph 实例 */
  function getGraph() { return graph }

  /** 根据客户端坐标获取节点索引 */
  function getNodeAtPoint(clientX: number, clientY: number): number | null {
    if (!graph) return null
    const p = graph.pageToLocal(clientX + window.scrollX, clientY + window.scrollY)
    const nodes = graph.getNodesFromPoint(p.x, p.y)
    if (nodes.length === 0) return null
    const meta = nodes[0].getData<X6NodeMeta>()
    return meta?.index ?? null
  }

  /** 缩放控制 */
  function zoomIn() { graph?.zoom(0.2) }
  function zoomOut() { graph?.zoom(-0.2) }
  function zoomToFitView() { graph?.zoomToFit({ maxScale: 2, padding: 40 }) }

  watch(() => characters.value, () => nextTick(scheduleRebuild), { deep: true })
  watch(() => relationships.value, () => nextTick(scheduleRebuild), { deep: true })

  onMounted(() => {
    if (!containerRef.value) return
    if (!graphBuilt) {
      const rect = containerRef.value.getBoundingClientRect()
      if (rect.width > 0 && rect.height > 0) ensureGraphBuilt()
    }
    // 观察容器尺寸变化
    resizeObserver = new ResizeObserver(() => {
      if (!containerRef.value) return
      const rect = containerRef.value.getBoundingClientRect()
      const hasValidSize = rect.width > 0 && rect.height > 0

      if (!graphBuilt) {
        if (hasValidSize) ensureGraphBuilt()
        return
      }

      // 容器尺寸有效时才调度 relayout
      if (onContainerResize && hasValidSize) {
        if (resizeRelayoutTimer) clearTimeout(resizeRelayoutTimer)
        resizeRelayoutTimer = setTimeout(() => {
          onContainerResize?.()
        }, 350)
      }
    })
    resizeObserver.observe(containerRef.value)
  })

  onUnmounted(() => {
    if (graph) graph.dispose()
    if (resizeObserver) resizeObserver.disconnect()
    if (rebuildTimer) clearTimeout(rebuildTimer)
    if (resizeRelayoutTimer) clearTimeout(resizeRelayoutTimer)
  })

  return {
    graphReady,
    getGraph,
    getNodeAtPoint,
    setOnNodeClick,
		setOnNodeSingleClick,
		setOnBlankClick,
    setOnEdgeConnected,
    setOnAfterBuild,
    setOnContainerResize,
    ensureGraphBuilt,
    zoomIn,
    zoomOut,
    zoomToFitView,
  }
}
