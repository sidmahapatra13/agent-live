import { useEffect, useRef, useCallback } from 'react'
import * as d3Force from 'd3-force'
import { select } from 'd3-selection'
import 'd3-transition' // registers .transition() on d3-selection

// ── Types ────────────────────────────────────────────────

export type NodeDef = {
  id: string
  label: string
  kind: 'file' | 'command' | 'thought'
  event_type: string
}

export type EdgeDef = {
  source: string
  target: string
  kind: 'read' | 'write' | 'exec'
}

// Extended node type used internally by the simulation
type SimNode = d3Force.SimulationNodeDatum & NodeDef

type SimLink = d3Force.SimulationLinkDatum<SimNode> & {
  kind: string
}

// ── Constants ─────────────────────────────────────────────

const NODE_COLORS: Record<string, string> = {
  file_read: '#3b82f6',
  file_write: '#22c55e',
  command: '#eab308',
  thought: '#a855f7',
  plan_step: '#06b6d4',
}

const EDGE_COLORS: Record<string, string> = {
  read: '#3b82f6',
  write: '#22c55e',
  exec: '#eab308',
}

const NODE_RADIUS: Record<string, number> = {
  file: 6,
  command: 8,
  thought: 5,
}

const AGENT_COLOR = '#60a5fa'

// ── Component ─────────────────────────────────────────────

type Props = {
  nodes: NodeDef[]
  edges: EdgeDef[]
  agentPosition: { source: string; target: string } | null
}

export default function GraphCanvas({ nodes, edges, agentPosition }: Props) {
  const svgRef = useRef<SVGSVGElement>(null)
  const simRef = useRef<d3Force.Simulation<SimNode, SimLink> | null>(null)
  const nodesRef = useRef<SimNode[]>([])
  const linksRef = useRef<SimLink[]>([])
  const widthRef = useRef(800)
  const heightRef = useRef(600)

  // Agent particle animation state
  const agentPos = useRef({ x: 100, y: 100 })
  const agentTarget = useRef({ x: 100, y: 100 })

  // ── Initialize SVG DOM and force simulation ──────────
  useEffect(() => {
    const svgEl = svgRef.current
    if (!svgEl) return

    const svg = select(svgEl)
    svg.selectAll('*').remove()

    // Root group
    const root = svg.append('g').attr('class', 'root')

    // Defs for filters and patterns
    const defs = svg.append('defs')

    // Subtle dot grid background pattern
    const pattern = defs
      .append('pattern')
      .attr('id', 'grid')
      .attr('width', 40)
      .attr('height', 40)
      .attr('patternUnits', 'userSpaceOnUse')
    pattern
      .append('circle')
      .attr('cx', 20)
      .attr('cy', 20)
      .attr('r', 1)
      .attr('fill', '#1e293b')

    svg
      .append('rect')
      .attr('width', '100%')
      .attr('height', '100%')
      .attr('fill', 'url(#grid)')

    // Hover styles
    defs
      .append('style')
      .text(`
        .edge-line { transition: stroke-opacity 0.2s, stroke-width 0.2s; }
        .edge-line:hover { stroke-opacity: 0.9 !important; stroke-width: 3 !important; }
        .node-circle { transition: r 0.3s; cursor: pointer; }
        .node-circle:hover { r: 12 !important; }
        .node-label { pointer-events: none; }
      `)

    // Glow filter for nodes
    const filter = defs
      .append('filter')
      .attr('id', 'glow')
      .attr('x', '-50%')
      .attr('y', '-50%')
      .attr('width', '200%')
      .attr('height', '200%')
    filter.append('feGaussianBlur').attr('stdDeviation', '3').attr('result', 'blur')
    const merge = filter.append('feMerge')
    merge.append('feMergeNode').attr('in', 'blur')
    merge.append('feMergeNode').attr('in', 'SourceGraphic')

    // Stronger glow for agent
    const agentFilter = defs
      .append('filter')
      .attr('id', 'agent-glow')
      .attr('x', '-100%')
      .attr('y', '-100%')
      .attr('width', '300%')
      .attr('height', '300%')
    agentFilter.append('feGaussianBlur').attr('stdDeviation', '5').attr('result', 'blur')
    const agentMerge = agentFilter.append('feMerge')
    agentMerge.append('feMergeNode').attr('in', 'blur')
    agentMerge.append('feMergeNode').attr('in', 'SourceGraphic')

    // Static element groups (order matters for z-index)
    const edgeGroup = root.append('g').attr('class', 'edges')
    const nodeGroup = root.append('g').attr('class', 'nodes')
    const labelGroup = root.append('g').attr('class', 'labels')
    const agentGroup = root.append('g').attr('class', 'agent')

    // ── Simulation ─────────────────────────────────────
    const sim = d3Force
      .forceSimulation<SimNode>(nodesRef.current)
      .force(
        'link',
        d3Force
          .forceLink<SimNode, SimLink>(linksRef.current)
          .id((d) => d.id)
          .distance(120)
          .strength(0.3),
      )
      .force('charge', d3Force.forceManyBody().strength(-200))
      .force('center', d3Force.forceCenter(widthRef.current / 2, heightRef.current / 2))
      .force('collision', d3Force.forceCollide().radius(25))
      .alphaDecay(0.02)

    simRef.current = sim

    // ── Tick: update DOM positions ─────────────────────
    sim.on('tick', () => {
      // Edges
      const edgeSel = edgeGroup
        .selectAll<SVGLineElement, SimLink>('line')
        .data(linksRef.current, (d: SimLink) => {
          const sId = typeof d.source === 'string' ? d.source : (d.source as SimNode).id
          const tId = typeof d.target === 'string' ? d.target : (d.target as SimNode).id
          return `${sId}-${tId}`
        })

      edgeSel
        .join('line')
        .attr('class', 'edge-line')
        .attr('stroke', (d: SimLink) => EDGE_COLORS[d.kind] || '#374151')
        .attr('stroke-width', 1.5)
        .attr('stroke-opacity', 0.4)
        .attr('x1', (d: SimLink): number => {
          const s = d.source
          return typeof s === 'string' ? 0 : (s as SimNode).x ?? 0
        })
        .attr('y1', (d: SimLink): number => {
          const s = d.source
          return typeof s === 'string' ? 0 : (s as SimNode).y ?? 0
        })
        .attr('x2', (d: SimLink): number => {
          const t = d.target
          return typeof t === 'string' ? 0 : (t as SimNode).x ?? 0
        })
        .attr('y2', (d: SimLink): number => {
          const t = d.target
          return typeof t === 'string' ? 0 : (t as SimNode).y ?? 0
        })

      // Nodes
      const nodeSel = nodeGroup
        .selectAll<SVGCircleElement, SimNode>('circle')
        .data(nodesRef.current, (d: SimNode) => d.id)

      nodeSel
        .join(
          (enter) =>
            enter
              .append('circle')
              .attr('class', 'node-circle')
              .attr('r', 0)
              .attr('fill', (d: SimNode) => NODE_COLORS[d.event_type] || '#6b7280')
              .attr('filter', 'url(#glow)')
              .call((sel) =>
                sel
                  .transition()
                  .duration(400)
                  .attr('r', (d: SimNode) => NODE_RADIUS[d.kind] || 5),
              ),
          (update) =>
            update.call((sel) =>
              sel
                .transition()
                .duration(300)
                .attr('fill', (d: SimNode) => NODE_COLORS[d.event_type] || '#6b7280'),
            ),
        )
        .attr('cx', (d: SimNode): number => d.x ?? 0)
        .attr('cy', (d: SimNode): number => d.y ?? 0)
        .attr('title', (d: SimNode) => d.label)

      // Labels
      const labelSel = labelGroup
        .selectAll<SVGTextElement, SimNode>('text')
        .data(nodesRef.current, (d: SimNode) => d.id)

      labelSel
        .join(
          (enter) =>
            enter
              .append('text')
              .attr('opacity', 0)
              .call((sel) =>
                sel.transition().duration(500).attr('opacity', 1),
              ),
        )
        .text((d: SimNode) => (d.label.length > 22 ? d.label.slice(0, 19) + '...' : d.label))
        .attr('x', (d: SimNode): number => d.x ?? 0)
        .attr('y', (d: SimNode): number => (d.y ?? 0) - (NODE_RADIUS[d.kind] || 5) - 4)
        .attr('text-anchor', 'middle')
        .attr('fill', '#9ca3af')
        .attr('font-size', '10px')
        .attr('font-family', "'SF Mono', 'Fira Code', monospace")

      // Agent particle — direct ref approach for a single element
      let agentCircle = agentGroup.select<SVGCircleElement>('circle').node()
      if (!agentCircle) {
        agentCircle = agentGroup
          .append('circle')
          .attr('r', 6)
          .attr('fill', AGENT_COLOR)
          .attr('filter', 'url(#agent-glow)')
          .attr('opacity', 0.9)
          .node()!
      }
      select(agentCircle)
        .attr('cx', agentPos.current.x)
        .attr('cy', agentPos.current.y)
    })

    return () => {
      sim.stop()
    }
  }, [])

  // ── Update simulation data when graph props change ───
  useEffect(() => {
    const sim = simRef.current
    if (!sim) return

    const currentIds = new Set(nodesRef.current.map((n) => n.id))
    const hasNew = nodes.some((n) => !currentIds.has(n.id))
    if (!hasNew && nodesRef.current.length === nodes.length) return

    // Preserve existing positions, assign random initial for new nodes
    const posMap = new Map(nodesRef.current.map((n) => [n.id, { x: n.x, y: n.y }]))

    const updatedNodes: SimNode[] = nodes.map((n) => {
      const pos = posMap.get(n.id)
      return {
        ...n,
        x: pos?.x ?? widthRef.current / 2 + (Math.random() - 0.5) * 300,
        y: pos?.y ?? heightRef.current / 2 + (Math.random() - 0.5) * 300,
        vx: 0,
        vy: 0,
        index: 0,
      }
    })

    // Add fixed agent node so link force can resolve __agent__ references
    const agentNodeId = '__agent__'
    if (updatedNodes.length > 0 && !updatedNodes.find((n) => n.id === agentNodeId)) {
      updatedNodes.unshift({
        id: agentNodeId,
        label: 'Agent',
        kind: 'command' as const,
        event_type: 'command',
        x: widthRef.current / 2,
        y: heightRef.current / 2,
        fx: widthRef.current / 2,
        fy: heightRef.current / 2,
      } as SimNode)
    }

    const updatedLinks: SimLink[] = edges.map((e) => ({
      source: e.source,
      target: e.target,
      kind: e.kind,
    }))

    nodesRef.current = updatedNodes
    linksRef.current = updatedLinks

    sim.nodes(updatedNodes)
    const linkForce = sim.force('link') as
      | d3Force.ForceLink<SimNode, SimLink>
      | undefined
    if (linkForce) {
      linkForce.links(updatedLinks)
    }
    sim.alpha(0.5).restart()
  }, [nodes, edges])

  // ── Update agent target position ─────────────────────
  useEffect(() => {
    if (!agentPosition) return

    // Find the target node's current sim position
    const tgt = nodesRef.current.find((n) => n.id === agentPosition.target)
    if (tgt) {
      agentTarget.current = { x: tgt.x as number, y: tgt.y as number }
    }
  }, [agentPosition])

  // ── Animation loop for agent particle ────────────────
  useEffect(() => {
    let raf: number
    const animate = () => {
      const dx = agentTarget.current.x - agentPos.current.x
      const dy = agentTarget.current.y - agentPos.current.y
      if (Math.abs(dx) > 0.5 || Math.abs(dy) > 0.5) {
        agentPos.current.x += dx * 0.08
        agentPos.current.y += dy * 0.08
      }
      raf = requestAnimationFrame(animate)
    }
    raf = requestAnimationFrame(animate)
    return () => cancelAnimationFrame(raf)
  }, [])

  // ── Responsive resize ────────────────────────────────
  const containerRef = useCallback((el: HTMLDivElement | null) => {
    if (!el) return
    const ro = new ResizeObserver(() => {
      const w = el.clientWidth
      const h = el.clientHeight
      if (w > 0 && h > 0) {
        widthRef.current = w
        heightRef.current = h
        const sim = simRef.current
        if (sim) {
          const center = sim.force('center') as
            | d3Force.ForceCenter<SimNode>
            | undefined
          if (center) {
            center.x(w / 2).y(h / 2)
          }
          sim.alpha(0.1).restart()
        }
      }
    })
    ro.observe(el)
  }, [])

  return (
    <div ref={containerRef} style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
      <svg
        ref={svgRef}
        width="100%"
        height="100%"
        style={{ background: '#0a0e17', display: 'block' }}
      />
    </div>
  )
}
