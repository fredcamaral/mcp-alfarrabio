'use client'

import { useEffect, useRef, useState, useMemo } from 'react'
import dynamic from 'next/dynamic'
import { useQuery } from '@apollo/client'
import { gql } from '@apollo/client'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Slider } from '@/components/ui/slider'
import { Label } from '@/components/ui/label'
import {
  Network,
  Maximize2,
  Minimize2,
  RotateCcw,
  Download,
  Settings,
  Info,
  ZoomIn,
  ZoomOut
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { ChunkType, RelationType } from '@/types/memory'

// Type for ForceGraph2D ref methods
interface ForceGraphMethods {
  d3Force: (forceName: string, force?: unknown) => unknown
  d3ReheatSimulation: () => void
  emitParticle: (link: unknown) => void
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

// Dynamically import ForceGraph2D to avoid SSR issues
const ForceGraph2D = dynamic(() => import('react-force-graph-2d'), {
  ssr: false,
  loading: () => <div className="flex items-center justify-center h-full">Loading graph...</div>
})

// GraphQL query for chunks and related data
const GET_GRAPH_DATA = gql`
  query GetGraphData($repository: String!, $limit: Int) {
    listChunks(repository: $repository, limit: $limit) {
      id
      type
      content
      summary
      timestamp
      tags
      sessionId
      toolsUsed
      concepts
      entities
    }
  }
`

interface GraphNode {
  id: string
  name: string
  type: ChunkType
  val: number
  color?: string
  content?: string
  summary?: string
  timestamp?: string
  confidence?: number
  tags?: string[]
}

interface GraphLink {
  source: string
  target: string
  type: RelationType
  confidence: number
  color?: string
  width?: number
}

// Interface for chunk data from GraphQL - matches ConversationChunk schema
interface ChunkData {
  id: string
  type: string // GraphQL returns string, we'll cast to ChunkType if valid
  content: string
  summary?: string
  timestamp: string
  tags?: string[]
  sessionId: string
  toolsUsed?: string[]
  concepts?: string[]
  entities?: string[]
}

interface GraphQLData {
  listChunks?: ChunkData[]
}

interface KnowledgeGraphProps {
  repository?: string
  className?: string
}

// Type colors mapping
const typeColors: Record<ChunkType, string> = {
  problem: '#ef4444',
  solution: '#10b981',
  architecture_decision: '#6366f1',
  session_summary: '#f59e0b',
  code_change: '#8b5cf6',
  discussion: '#06b6d4',
  analysis: '#ec4899',
  verification: '#14b8a6',
  question: '#f97316'
}

// Relation type colors
const relationColors: Record<RelationType, string> = {
  led_to: '#10b981',
  solved_by: '#22c55e',
  depends_on: '#f97316',
  enables: '#3b82f6',
  conflicts_with: '#ef4444',
  supersedes: '#a855f7',
  related_to: '#6b7280',
  follows_up: '#0ea5e9',
  precedes: '#0891b2',
  learned_from: '#8b5cf6',
  teaches: '#d946ef',
  exemplifies: '#ec4899',
  referenced_by: '#94a3b8',
  references: '#64748b'
}

export function KnowledgeGraph({ repository = process.env.NEXT_PUBLIC_DEFAULT_REPOSITORY || 'github.com/lerianstudio/lerian-mcp-memory', className }: KnowledgeGraphProps) {
  const graphRef = useRef<ForceGraphMethods>(null)
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null)
  const [selectedLink, setSelectedLink] = useState<GraphLink | null>(null)
  const [graphData, setGraphData] = useState<{ nodes: GraphNode[], links: GraphLink[] }>({ nodes: [], links: [] })
  const [isFullscreen, setIsFullscreen] = useState(false)
  
  // Graph controls
  const [minConfidence, setMinConfidence] = useState(0.5)
  const [selectedTypes] = useState<ChunkType[]>([])
  const [selectedRelations] = useState<RelationType[]>([])
  const [nodeSize, setNodeSize] = useState(1)
  const [showLabels, setShowLabels] = useState(true)
  
  // Fetch data
  const { data, loading, error, refetch } = useQuery<GraphQLData>(GET_GRAPH_DATA, {
    variables: { repository, limit: 200 }
  })

  // Process graph data
  useEffect(() => {
    if (!data) return

    const nodes: Map<string, GraphNode> = new Map()
    const links: GraphLink[] = []

    // Add nodes from chunks
    data.listChunks?.forEach((chunk: ChunkData) => {
      // Cast type to ChunkType if it's valid, otherwise use 'discussion' as fallback
      const chunkType = Object.keys(typeColors).includes(chunk.type) 
        ? chunk.type as ChunkType 
        : 'discussion' as ChunkType

      nodes.set(chunk.id, {
        id: chunk.id,
        name: chunk.summary || chunk.content.substring(0, 50) + '...',
        type: chunkType,
        val: 6, // Fixed node size since we don't have confidence scores
        color: typeColors[chunkType] || typeColors.discussion,
        content: chunk.content,
        summary: chunk.summary,
        timestamp: chunk.timestamp,
        confidence: undefined, // No confidence data available from listChunks
        tags: chunk.tags || []
      })
    })

    // Create links based on shared attributes and temporal relationships
    const chunks = data.listChunks || []
    
    // Create links based on shared concepts, entities, or tags
    chunks.forEach((chunk: ChunkData) => {
      chunks.forEach((otherChunk: ChunkData) => {
        if (chunk.id === otherChunk.id) return
        
        // Link chunks with shared concepts
        const sharedConcepts = (chunk.concepts || []).filter(concept => 
          (otherChunk.concepts || []).includes(concept)
        )
        
        // Link chunks with shared entities
        const sharedEntities = (chunk.entities || []).filter(entity => 
          (otherChunk.entities || []).includes(entity)
        )
        
        // Link chunks with shared tags
        const sharedTags = (chunk.tags || []).filter(tag => 
          (otherChunk.tags || []).includes(tag)
        )
        
        // Create links based on shared attributes
        if (sharedConcepts.length > 0 || sharedEntities.length > 1 || sharedTags.length > 1) {
          const totalShared = sharedConcepts.length + sharedEntities.length + sharedTags.length
          const confidence = Math.min(0.9, 0.4 + (totalShared * 0.1))
          
          // Avoid duplicate links
          const existingLink = links.find(link => 
            (link.source === chunk.id && link.target === otherChunk.id) ||
            (link.source === otherChunk.id && link.target === chunk.id)
          )
          
          if (!existingLink) {
            links.push({
              source: chunk.id,
              target: otherChunk.id,
              type: 'related_to' as RelationType,
              confidence,
              color: relationColors.related_to,
              width: Math.max(1, totalShared * 0.5)
            })
          }
        }
      })
    })
    
    // Create temporal links between consecutive chunks
    const sortedChunks = [...chunks].sort((a, b) => 
      new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    )
    
    for (let i = 0; i < sortedChunks.length - 1; i++) {
      const current = sortedChunks[i]
      const next = sortedChunks[i + 1]
      
      // Link if within 30 minutes
      const timeDiff = new Date(next.timestamp).getTime() - new Date(current.timestamp).getTime()
      if (timeDiff < 30 * 60 * 1000) {
        links.push({
          source: current.id,
          target: next.id,
          type: 'follows_up' as RelationType,
          confidence: 0.6,
          color: relationColors.follows_up,
          width: 1.5
        })
      }
    }
    
    // Create links between problems and solutions
    const problems = chunks.filter((c: ChunkData) => c.type === 'problem')
    const solutions = chunks.filter((c: ChunkData) => c.type === 'solution')
    
    problems.forEach((problem: ChunkData) => {
      solutions.forEach((solution: ChunkData) => {
        // Link if solution comes after problem and within 2 hours
        const problemTime = new Date(problem.timestamp).getTime()
        const solutionTime = new Date(solution.timestamp).getTime()
        if (solutionTime > problemTime && solutionTime - problemTime < 2 * 60 * 60 * 1000) {
          links.push({
            source: problem.id,
            target: solution.id,
            type: 'solved_by' as RelationType,
            confidence: 0.8,
            color: relationColors.solved_by,
            width: 3
          })
        }
      })
    })

    // Filter based on controls
    const filteredLinks = links.filter(link => {
      if (link.confidence < minConfidence) return false
      if (selectedRelations.length > 0 && !selectedRelations.includes(link.type)) return false
      return true
    })

    const connectedNodeIds = new Set<string>()
    filteredLinks.forEach(link => {
      connectedNodeIds.add(link.source)
      connectedNodeIds.add(link.target)
    })

    // Include all nodes, not just connected ones, so isolated nodes are still visible
    const filteredNodes = Array.from(nodes.values()).filter(node => {
      if (selectedTypes.length > 0 && !selectedTypes.includes(node.type)) return false
      return true
    })

    setGraphData({
      nodes: filteredNodes,
      links: filteredLinks
    })
  }, [data, minConfidence, selectedTypes, selectedRelations])

  // Graph configuration
  const graphConfig = useMemo(() => ({
    nodeRelSize: nodeSize * 5,
    nodeLabel: showLabels ? 'name' : undefined,
    nodeCanvasObject: (node: unknown, ctx: CanvasRenderingContext2D, globalScale: number) => {
      const graphNode = node as GraphNode & { x: number; y: number }
      const label = graphNode.name
      const fontSize = 12 / globalScale
      ctx.font = `${fontSize}px Sans-Serif`
      
      // Draw node circle
      ctx.beginPath()
      ctx.arc(graphNode.x, graphNode.y, graphNode.val * nodeSize, 0, 2 * Math.PI, false)
      ctx.fillStyle = graphNode.color || '#999'
      ctx.fill()
      
      // Draw label if enabled
      if (showLabels && globalScale > 0.5) {
        ctx.textAlign = 'center'
        ctx.textBaseline = 'middle'
        ctx.fillStyle = '#fff'
        ctx.fillText(label.substring(0, 20), graphNode.x, graphNode.y)
      }
    },
    linkDirectionalArrowLength: 3.5,
    linkDirectionalArrowRelPos: 1,
    linkCurvature: 0.25,
    linkWidth: (link: unknown) => (link as GraphLink).width || 1,
    linkColor: (link: unknown) => (link as GraphLink).color || '#999',
    d3AlphaDecay: 0.01,
    d3VelocityDecay: 0.3,
    warmupTicks: 100,
    cooldownTime: 15000,
    onNodeClick: (node: unknown) => setSelectedNode(node as GraphNode),
    onLinkClick: (link: unknown) => setSelectedLink(link as GraphLink),
    onBackgroundClick: () => {
      setSelectedNode(null)
      setSelectedLink(null)
    }
  }), [nodeSize, showLabels])

  const handleZoomIn = () => {
    if (graphRef.current) {
      graphRef.current.zoom(1.5, 300)
    }
  }

  const handleZoomOut = () => {
    if (graphRef.current) {
      graphRef.current.zoom(0.75, 300)
    }
  }

  const handleReset = () => {
    if (graphRef.current) {
      graphRef.current.zoomToFit(400)
    }
  }

  const handleDownload = () => {
    if (graphRef.current) {
      const canvas = graphRef.current.screen2Canvas()
      if (canvas) {
        const link = document.createElement('a')
        link.download = 'knowledge-graph.png'
        link.href = canvas.toDataURL()
        link.click()
      }
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center space-y-4">
          <Network className="h-12 w-12 animate-pulse mx-auto" />
          <p>Loading knowledge graph...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="text-center space-y-4">
          <Network className="h-12 w-12 text-destructive mx-auto" />
          <p className="text-destructive">Failed to load graph data</p>
          <Button onClick={() => refetch()}>Retry</Button>
        </div>
      </div>
    )
  }

  return (
    <div className={cn("relative", isFullscreen && "fixed inset-0 z-50 bg-background", className)}>
      <Card className={cn("h-full", isFullscreen && "rounded-none border-0")}>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-4">
          <CardTitle className="flex items-center gap-2">
            <Network className="h-5 w-5" />
            Knowledge Graph
          </CardTitle>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleZoomIn}
            >
              <ZoomIn className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleZoomOut}
            >
              <ZoomOut className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleReset}
            >
              <RotateCcw className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleDownload}
            >
              <Download className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsFullscreen(!isFullscreen)}
            >
              {isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
            </Button>
          </div>
        </CardHeader>
        <CardContent className="p-0 h-[calc(100%-5rem)] relative">
          {/* Graph */}
          <div className="absolute inset-0">
            <ForceGraph2D
              ref={graphRef}
              graphData={graphData}
              {...graphConfig}
              width={isFullscreen ? window.innerWidth : undefined}
              height={isFullscreen ? window.innerHeight - 80 : undefined}
            />
          </div>

          {/* Controls Panel */}
          <div className="absolute top-4 left-4 w-80 space-y-4">
            <Card className="bg-background/95 backdrop-blur">
              <CardHeader className="pb-3">
                <h3 className="text-sm font-medium flex items-center gap-2">
                  <Settings className="h-4 w-4" />
                  Graph Controls
                </h3>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label className="text-xs">Min Confidence: {Math.round(minConfidence * 100)}%</Label>
                  <Slider
                    value={[minConfidence]}
                    onValueChange={([value]) => setMinConfidence(value)}
                    min={0}
                    max={1}
                    step={0.1}
                    className="w-full"
                  />
                </div>

                <div className="space-y-2">
                  <Label className="text-xs">Node Size</Label>
                  <Slider
                    value={[nodeSize]}
                    onValueChange={([value]) => setNodeSize(value)}
                    min={0.5}
                    max={3}
                    step={0.1}
                    className="w-full"
                  />
                </div>


                <div className="flex items-center justify-between">
                  <Label className="text-xs">Show Labels</Label>
                  <Button
                    variant={showLabels ? "default" : "outline"}
                    size="sm"
                    onClick={() => setShowLabels(!showLabels)}
                  >
                    {showLabels ? "On" : "Off"}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Selected Node/Link Info */}
          {(selectedNode || selectedLink) && (
            <div className="absolute bottom-4 right-4 w-96">
              <Card className="bg-background/95 backdrop-blur">
                <CardHeader className="pb-3">
                  <h3 className="text-sm font-medium flex items-center gap-2">
                    <Info className="h-4 w-4" />
                    {selectedNode ? 'Node Details' : 'Link Details'}
                  </h3>
                </CardHeader>
                <CardContent className="space-y-3">
                  {selectedNode && (
                    <>
                      <div>
                        <Badge 
                          className="mb-2" 
                          style={{ backgroundColor: typeColors[selectedNode.type] }}
                        >
                          {selectedNode.type.replace('_', ' ')}
                        </Badge>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">Summary</p>
                        <p className="text-sm">{selectedNode.summary || 'No summary'}</p>
                      </div>
                      {selectedNode.confidence !== undefined && (
                        <div>
                          <p className="text-xs text-muted-foreground">Confidence</p>
                          <p className="text-sm">{Math.round(selectedNode.confidence * 100)}%</p>
                        </div>
                      )}
                      {selectedNode.tags && selectedNode.tags.length > 0 && (
                        <div>
                          <p className="text-xs text-muted-foreground mb-1">Tags</p>
                          <div className="flex flex-wrap gap-1">
                            {selectedNode.tags.map(tag => (
                              <Badge key={tag} variant="secondary" className="text-xs">
                                {tag}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      )}
                    </>
                  )}
                  {selectedLink && (
                    <>
                      <div>
                        <Badge 
                          className="mb-2" 
                          style={{ backgroundColor: relationColors[selectedLink.type] }}
                        >
                          {selectedLink.type.replace('_', ' ')}
                        </Badge>
                      </div>
                      <div>
                        <p className="text-xs text-muted-foreground">Confidence</p>
                        <p className="text-sm">{Math.round(selectedLink.confidence * 100)}%</p>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>
            </div>
          )}

          {/* Graph Stats */}
          <div className="absolute top-4 right-4">
            <Card className="bg-background/95 backdrop-blur">
              <CardContent className="p-3">
                <div className="text-xs space-y-1">
                  <p>Nodes: {graphData.nodes.length}</p>
                  <p>Links: {graphData.links.length}</p>
                </div>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}