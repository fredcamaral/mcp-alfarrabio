declare module 'react-force-graph-2d' {
  import { ForwardRefExoticComponent, RefAttributes } from 'react'

  interface NodeObject {
    id?: string | number
    x?: number
    y?: number
    vx?: number
    vy?: number
    fx?: number | null
    fy?: number | null
    [key: string]: any
  }

  interface LinkObject {
    source?: string | number | NodeObject
    target?: string | number | NodeObject
    [key: string]: any
  }

  interface GraphData {
    nodes: NodeObject[]
    links: LinkObject[]
  }

  interface ForceGraphMethods {
    d3Force: (forceName: string, force?: any) => any
    d3ReheatSimulation: () => void
    emitParticle: (link: LinkObject) => void
    pauseAnimation: () => void
    resumeAnimation: () => void
    centerAt: (x?: number, y?: number, duration?: number) => void
    zoom: (zoomLevel: number, duration?: number) => void
    zoomToFit: (duration?: number, padding?: number) => void
    getGraphBbox: () => { x: [number, number]; y: [number, number] }
    screen2GraphCoords: (x: number, y: number) => { x: number; y: number }
    graph2ScreenCoords: (x: number, y: number) => { x: number; y: number }
    screen2Canvas: (x?: number, y?: number) => HTMLCanvasElement | null
  }

  interface ForceGraphProps {
    graphData?: GraphData
    width?: number
    height?: number
    backgroundColor?: string
    nodeRelSize?: number
    nodeId?: string | ((node: NodeObject) => string)
    nodeLabel?: string | ((node: NodeObject) => string) | null
    nodeVal?: number | string | ((node: NodeObject) => number)
    nodeVisibility?: boolean | string | ((node: NodeObject) => boolean)
    nodeColor?: string | ((node: NodeObject) => string)
    nodeAutoColorBy?: string | ((node: NodeObject) => string | null)
    nodeCanvasObject?: (node: NodeObject, ctx: CanvasRenderingContext2D, globalScale: number) => void
    nodeCanvasObjectMode?: string | ((node: NodeObject) => string)
    linkSource?: string | ((link: LinkObject) => string)
    linkTarget?: string | ((link: LinkObject) => string)
    linkLabel?: string | ((link: LinkObject) => string) | null
    linkVisibility?: boolean | string | ((link: LinkObject) => boolean)
    linkColor?: string | ((link: LinkObject) => string)
    linkAutoColorBy?: string | ((link: LinkObject) => string | null)
    linkWidth?: number | string | ((link: LinkObject) => number)
    linkCurvature?: number | string | ((link: LinkObject) => number)
    linkCanvasObject?: (link: LinkObject, ctx: CanvasRenderingContext2D, globalScale: number) => void
    linkCanvasObjectMode?: string | ((link: LinkObject) => string)
    linkDirectionalArrowLength?: number | string | ((link: LinkObject) => number)
    linkDirectionalArrowColor?: string | ((link: LinkObject) => string)
    linkDirectionalArrowRelPos?: number | string | ((link: LinkObject) => number)
    linkDirectionalParticles?: number | string | ((link: LinkObject) => number)
    linkDirectionalParticleSpeed?: number | string | ((link: LinkObject) => number)
    linkDirectionalParticleWidth?: number | string | ((link: LinkObject) => number)
    linkDirectionalParticleColor?: string | ((link: LinkObject) => string)
    dagMode?: 'td' | 'bu' | 'lr' | 'rl' | 'radialout' | 'radialin' | null
    dagLevelDistance?: number
    dagNodeFilter?: (node: NodeObject) => boolean
    onDagError?: (loopNodeIds: (string | number)[]) => void
    d3AlphaMin?: number
    d3AlphaDecay?: number
    d3VelocityDecay?: number
    warmupTicks?: number
    cooldownTicks?: number
    cooldownTime?: number
    onEngineTick?: () => void
    onEngineStop?: () => void
    onNodeClick?: (node: NodeObject, event: MouseEvent) => void
    onNodeRightClick?: (node: NodeObject, event: MouseEvent) => void
    onNodeHover?: (node: NodeObject | null, previousNode: NodeObject | null) => void
    onNodeDrag?: (node: NodeObject, translate: { x: number; y: number }) => void
    onNodeDragEnd?: (node: NodeObject, translate: { x: number; y: number }) => void
    onLinkClick?: (link: LinkObject, event: MouseEvent) => void
    onLinkRightClick?: (link: LinkObject, event: MouseEvent) => void
    onLinkHover?: (link: LinkObject | null, previousLink: LinkObject | null) => void
    onBackgroundClick?: (event: MouseEvent) => void
    onBackgroundRightClick?: (event: MouseEvent) => void
    linkHoverPrecision?: number
    onZoom?: (transform: { k: number; x: number; y: number }) => void
    onZoomEnd?: (transform: { k: number; x: number; y: number }) => void
    enableNodeDrag?: boolean
    enableZoomInteraction?: boolean
    enablePanInteraction?: boolean
    enablePointerInteraction?: boolean
  }

  const ForceGraph2D: ForwardRefExoticComponent<ForceGraphProps & RefAttributes<ForceGraphMethods>>
  export default ForceGraph2D
}