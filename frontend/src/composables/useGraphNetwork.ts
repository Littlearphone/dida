/**
 * vis-network 图谱生命周期 composable
 * 管理：网络构建、重建、ResizeObserver、重新布局
 */
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { Network } from 'vis-network'
import { DataSet } from 'vis-data'
import type { Character, NovelRelationship } from '../types'
import { buildNodes, buildEdges, DEFAULT_GRAPH_OPTIONS } from '../utils/graphUtils'

export function useGraphNetwork(
  containerRef: { value: HTMLDivElement | undefined },
  characters: { value: Character[] },
  relationships: { value: NovelRelationship[] | undefined },
  connectMode: { value: boolean },
  cancelConnect: () => void,
  enterConnectMode: () => void,
) {
  let network: any = null
  let graphBuilt = false
  let resizeObserver: ResizeObserver | null = null
  let resizeFitTimer: ReturnType<typeof setTimeout> | null = null
  let rebuildTimer: ReturnType<typeof setTimeout> | null = null

  const svgViewBox = ref('0 0 100 100')

  function updateSvgViewBox() {
    const el = containerRef.value
    if (!el) return
    const rect = el.getBoundingClientRect()
    svgViewBox.value = `0 0 ${Math.max(1, Math.round(rect.width))} ${Math.max(1, Math.round(rect.height))}`
  }

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
    const wasConnect = connectMode.value
    cancelConnect()

    if (network) { network.destroy(); network = null }

    const chars = characters.value
    if (chars.length === 0) return

    const nodeItems = buildNodes(chars)
    const edgeItems = buildEdges(chars, relationships.value)
    const nodes = new DataSet(nodeItems as any)
    const edges = new DataSet(edgeItems as any)

    network = new (Network as any)(
      containerRef.value,
      { nodes, edges } as any,
      DEFAULT_GRAPH_OPTIONS,
    )

    // 点击事件：编辑角色
    network.on('click', (params: any) => {
      if (connectMode.value) return
      if (params.nodes.length === 0) return
      const nodeIdx = params.nodes[0] as number
      onNodeClick(nodeIdx)
    })

    network.once('startStabilization', () => network?.fit({ animation: false }))
    network.once('stabilizationIterationsDone', () => {
      network?.fit({ animation: true })
      network?.setOptions({ physics: { enabled: false } })
      if (wasConnect) nextTick(enterConnectMode)
      nextTick(() => onAfterBuild?.())
    })

    nextTick(updateSvgViewBox)
  }

  let onNodeClick = (_nodeIdx: number) => {}

  function setOnNodeClick(cb: (idx: number) => void) {
    onNodeClick = cb
  }

  /** 构建完成回调（角色变更重建后恢复布局） */
  let onAfterBuild: (() => void) | null = null

  function setOnAfterBuild(cb: () => void) {
    onAfterBuild = cb
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
    if (rebuildTimer) clearTimeout(rebuildTimer)
    rebuildTimer = setTimeout(() => {
      if (containerRef.value) ensureGraphBuilt()
    }, 100)
  }

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

  function getNetwork() { return network }

  watch(() => characters.value, () => nextTick(scheduleRebuild), { deep: true })
  watch(() => relationships.value, () => nextTick(scheduleRebuild), { deep: true })

  onMounted(() => {
    if (!containerRef.value) return
    if (!graphBuilt) {
      const rect = containerRef.value.getBoundingClientRect()
      if (rect.width > 0 && rect.height > 0) ensureGraphBuilt()
    }
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.contentRect.width > 0 && entry.contentRect.height > 0) {
          if (!graphBuilt) ensureGraphBuilt()
          else scheduleResizeFit()
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

  return {
    svgViewBox,
    getNetwork,
    setOnNodeClick,
    setOnAfterBuild,
    ensureGraphBuilt,
    reLayout,
  }
}
