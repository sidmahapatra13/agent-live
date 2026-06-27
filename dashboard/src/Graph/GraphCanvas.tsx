import { useEffect, useRef, useCallback } from 'react'
import * as d3Force from 'd3-force'
import { select } from 'd3-selection'
import 'd3-transition'

// ── Types ─────────────────────────────────────────────────────

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

type SimNode = d3Force.SimulationNodeDatum & NodeDef
type SimLink = d3Force.SimulationLinkDatum<SimNode> & { kind: string }

// ── Palette ────────────────────────────────────────────────────

const C: Record<string, string> = {
  file_read:  '#60a5fa',
  file_write: '#34d399',
  command:    '#fbbf24',
  thought:    '#c084fc',
  plan_step:  '#22d3ee',
  __agent__:  '#60a5fa',
  bg:         '#070b14',
  grid:       '#131a2e',
}

const EDGE: Record<string, string> = {
  read:  '#3b82f6',
  write: '#22c55e',
  exec:  '#f59e0b',
}

const ICONS: Record<string, string> = {
  file_read:  '📖',
  file_write: '✏️',
  command:    '⚡',
  thought:    '💭',
  plan_step:  '🎯',
}

const NODE_R = { file: 22, command: 24, thought: 20 }
const AGENT_R = 28

// ── Edge key ──────────────────────────────────────────────────
function edgeKey(d: SimLink): string {
  const s = typeof d.source === 'string' ? d.source : (d.source as SimNode).id
  const t = typeof d.target === 'string' ? d.target : (d.target as SimNode).id
  return `${s}→${t}`
}

// ── Component ──────────────────────────────────────────────────

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
  const particlePos = useRef({ x: 200, y: 200 })
  const particleTarget = useRef({ x: 200, y: 200 })
  const hasParticleTarget = useRef(false)

  // ── Init ──
  useEffect(() => {
    const svgEl = svgRef.current
    if (!svgEl) return

    const svg = select(svgEl)
    svg.selectAll('*').remove()
    const root = svg.append('g').attr('class', 'root')
    const defs = svg.append('defs')

    // Grid
    const pat = defs.append('pattern').attr('id', 'g')
      .attr('width', 28).attr('height', 28).attr('patternUnits', 'userSpaceOnUse')
    pat.append('circle').attr('cx', 14).attr('cy', 14).attr('r', 0.7).attr('fill', C.grid)
    svg.append('rect').attr('width', '100%').attr('height', '100%').attr('fill', 'url(#g)')

    // Edge arrow markers
    for (const [k, color] of Object.entries(EDGE)) {
      defs.append('marker')
        .attr('id', `a-${k}`).attr('viewBox', '0 0 10 10')
        .attr('refX', 24).attr('refY', 5)
        .attr('markerWidth', 5).attr('markerHeight', 5).attr('orient', 'auto')
        .append('path').attr('d', 'M 0 0 L 10 5 L 0 10 z').attr('fill', color)
    }

    // Glow filter (standard)
    const glow = defs.append('filter')
      .attr('id', 'gl').attr('x', '-60%').attr('y', '-60%')
      .attr('width', '220%').attr('height', '220%')
    glow.append('feGaussianBlur').attr('stdDeviation', '3').attr('result', 'b')
    glow.append('feMerge').selectAll('feMergeNode')
      .data(['b', 'SourceGraphic']).join('feMergeNode').attr('in', d => d)

    // Strong glow (agent)
    const aglow = defs.append('filter')
      .attr('id', 'agl').attr('x', '-120%').attr('y', '-120%')
      .attr('width', '340%').attr('height', '340%')
    aglow.append('feGaussianBlur').attr('stdDeviation', '8').attr('result', 'b')
    aglow.append('feMerge').selectAll('feMergeNode')
      .data(['b', 'SourceGraphic']).join('feMergeNode').attr('in', d => d)

    // CSS
    defs.append('style').text(`
      @keyframes pulse-halo {
        0%,100% { opacity:0.4; r:34; }
        50%      { opacity:0.15; r:40; }
      }
      .edge-line { transition: stroke-opacity .12s, stroke-width .12s; }
      .node-group { cursor:pointer; }
      .node-group .node-label-row { opacity:0.6; transition: opacity .15s; }
      .node-group:hover .node-label-row { opacity:1; }
      .node-group:hover .node-badge { filter:brightness(1.15); }
    `)

    // Z-groups
    const edgesG = root.append('g')
    const nodesG = root.append('g')
    const agentG = root.append('g')

    // Simulation
    const sim = d3Force.forceSimulation<SimNode>(nodesRef.current)
      .force('link', d3Force.forceLink<SimNode, SimLink>(linksRef.current)
        .id(d => d.id).distance(180).strength(0.15))
      .force('charge', d3Force.forceManyBody().strength(-500))
      .force('center', d3Force.forceCenter(widthRef.current / 2, heightRef.current / 2))
      .force('collision', d3Force.forceCollide().radius(50))
      .alphaDecay(0.014)

    simRef.current = sim

    function nr(d: SimNode): number { return d.id === '__agent__' ? AGENT_R : NODE_R[d.kind] || 22 }
    function nc(d: SimNode): string { return d.id === '__agent__' ? C.__agent__ : C[d.event_type] || '#6b7280' }

    // ── Tick ──
    sim.on('tick', () => {
      // Edges
      edgesG.selectAll<SVGLineElement, SimLink>('line')
        .data(linksRef.current, edgeKey)
        .join('line')
        .attr('class', 'edge-line')
        .attr('stroke', d => EDGE[d.kind] || '#374151')
        .attr('stroke-width', 1.8)
        .attr('stroke-opacity', 0.3)
        .attr('marker-end', d => `url(#a-${d.kind})`)
        .attr('x1', d => (typeof d.source === 'string' ? 0 : (d.source as SimNode).x ?? 0))
        .attr('y1', d => (typeof d.source === 'string' ? 0 : (d.source as SimNode).y ?? 0))
        .attr('x2', d => (typeof d.target === 'string' ? 0 : (d.target as SimNode).x ?? 0))
        .attr('y2', d => (typeof d.target === 'string' ? 0 : (d.target as SimNode).y ?? 0))

      // Nodes
      nodesG.selectAll<SVGGElement, SimNode>('g.n')
        .data(nodesRef.current, (d: SimNode) => d.id)
        .join(
          enter => {
            const g = enter.append('g').attr('class', 'n')

            // Outer ring
            g.append('circle').attr('class', 'h')
            // Main circle
            g.append('circle').attr('class', 'b')
            // Inner highlight dot
            g.append('circle').attr('class', 'i')
            // Emoji icon
            g.append('text').attr('class', 'ic')
              .attr('text-anchor', 'middle').attr('dominant-baseline', 'central')
              .attr('font-size', 12).attr('pointer-events', 'none')
              .attr('opacity', 0).call(s => s.transition().duration(250).attr('opacity', 0.85))

            // Label row
            const lr = g.append('g').attr('class', 'node-label-row')
            lr.append('rect').attr('class', 'lb')
              .attr('rx', 5).attr('ry', 5)
              .attr('fill', 'rgba(7,11,20,0.88)')
              .attr('stroke', 'rgba(148,163,184,0.13)').attr('stroke-width', 1)
            lr.append('text').attr('class', 'lt')
              .attr('text-anchor', 'middle')
              .attr('fill', '#f1f5f9')
              .attr('font-family', "'Inter', system-ui, sans-serif")
              .attr('font-size', 10).attr('font-weight', 500)
              .attr('pointer-events', 'none')

            // Entrance
            g.attr('opacity', 0).attr('transform', 'scale(0.3)')
              .call(s => s.transition().duration(400).ease(Math.sqrt as any)
                .attr('opacity', 1).attr('transform', 'scale(1)'))

            // Bring to center of actual transform later
            return g
          },
          update => update,
          exit => exit.call(s => s.transition().duration(200).attr('opacity', 0).remove()),
        )
        .attr('transform', d => `translate(${d.x ?? 0},${d.y ?? 0})`)
        .each(function (d: SimNode) {
          const g = select(this)
          const r = nr(d)
          const c = nc(d)
          const isAgent = d.id === '__agent__'

          // Halo ring
          g.select<SVGCircleElement>('circle.h')
            .attr('r', r + 8)
            .attr('fill', 'none').attr('stroke', c)
            .attr('stroke-width', 1.5)
            .attr('stroke-opacity', isAgent ? 0.5 : 0.2)
            .style('animation', isAgent ? 'pulse-halo 3s ease-in-out infinite' : 'none')

          // Badge
          g.select<SVGCircleElement>('circle.b')
            .attr('r', r).attr('fill', c)
            .attr('fill-opacity', isAgent ? 1 : 0.85)
            .attr('filter', 'url(#gl)')
            .attr('stroke', 'rgba(255,255,255,0.1)').attr('stroke-width', 1.2)

          // Inner highlight
          g.select<SVGCircleElement>('circle.i')
            .attr('r', r * 0.35)
            .attr('fill', 'rgba(255,255,255,0.13)')
            .attr('cx', -r * 0.25).attr('cy', -r * 0.25)

          // Icon
          const icon = isAgent ? '' : (ICONS[d.event_type] || ' ')
          g.select<SVGTextElement>('text.ic')
            .text(icon).attr('y', 0)

          // Label
          const label = isAgent ? 'Agent' : d.label
          const display = label.length > 28 ? label.slice(0, 25) + '…' : label
          const lt = g.select<SVGTextElement>('text.lt').text(display)

          const off = isAgent ? r + 18 : r + 15
          g.select('g.node-label-row').attr('transform', `translate(0,${-off})`)

          let tw: number
          try { tw = (lt.node() as SVGTextElement).getComputedTextLength() }
          catch { tw = display.length * 6 }
          const bw = Math.max(tw + 16, 26)
          g.select<SVGRectElement>('rect.lb')
            .attr('x', -bw / 2).attr('y', -9)
            .attr('width', bw).attr('height', 18)
        })

      // Agent particle — only render once we have a target
      if (hasParticleTarget.current) {
        let p = agentG.select<SVGCircleElement>('circle').node()
        if (!p) {
          p = agentG.append('circle')
            .attr('r', 3.5).attr('fill', '#93c5fd')
            .attr('filter', 'url(#agl)').attr('opacity', 0.8).node()!
        }
        select(p).attr('cx', particlePos.current.x).attr('cy', particlePos.current.y)
      } else {
        agentG.selectAll('circle').remove()
      }
    })

    return () => { sim.stop() }
  }, [])

  // ── Sync data ──
  useEffect(() => {
    const sim = simRef.current
    if (!sim) return
    const curIds = new Set(nodesRef.current.map(n => n.id))
    if (!nodes.some(n => !curIds.has(n.id)) && nodesRef.current.length === nodes.length) return

    const posMap = new Map(nodesRef.current.map(n => [n.id, { x: n.x, y: n.y }]))
    const updated: SimNode[] = nodes.map(n => {
      const pos = posMap.get(n.id)
      return { ...n, x: pos?.x ?? widthRef.current / 2 + (Math.random() - 0.5) * 400, y: pos?.y ?? heightRef.current / 2 + (Math.random() - 0.5) * 400, vx: 0, vy: 0, index: 0 }
    })

    const agentId = '__agent__'
    if (updated.length > 0 && !updated.find(n => n.id === agentId)) {
      updated.unshift({ id: agentId, label: 'Agent', kind: 'command' as const, event_type: 'command', x: widthRef.current / 2, y: heightRef.current / 2, fx: widthRef.current / 2, fy: heightRef.current / 2 } as SimNode)
    }

    nodesRef.current = updated
    linksRef.current = edges.map(e => ({ source: e.source, target: e.target, kind: e.kind }))
    sim.nodes(updated)
    const lf = sim.force('link') as d3Force.ForceLink<SimNode, SimLink> | undefined
    if (lf) lf.links(linksRef.current)
    sim.alpha(0.5).restart()
  }, [nodes, edges])

  useEffect(() => {
    if (!agentPosition) {
      hasParticleTarget.current = false
      return
    }
    hasParticleTarget.current = true
    const tgt = nodesRef.current.find(n => n.id === agentPosition.target)
    if (tgt) particleTarget.current = { x: tgt.x as number, y: tgt.y as number }
  }, [agentPosition])

  useEffect(() => {
    let raf: number
    const tick = () => {
      const dx = particleTarget.current.x - particlePos.current.x
      const dy = particleTarget.current.y - particlePos.current.y
      if (Math.abs(dx) > 0.5 || Math.abs(dy) > 0.5) {
        particlePos.current.x += dx * 0.08
        particlePos.current.y += dy * 0.08
      }
      raf = requestAnimationFrame(tick)
    }
    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  }, [])

  const containerRef = useCallback((el: HTMLDivElement | null) => {
    if (!el) return
    const ro = new ResizeObserver(() => {
      const w = el.clientWidth, h = el.clientHeight
      if (w <= 0 || h <= 0) return
      widthRef.current = w; heightRef.current = h
      const sim = simRef.current
      if (sim) {
        const c = sim.force('center') as d3Force.ForceCenter<SimNode> | undefined
        if (c) c.x(w / 2).y(h / 2)
        sim.alpha(0.1).restart()
      }
    })
    ro.observe(el)
  }, [])

  return (
    <div ref={containerRef} style={{ flex: 1, position: 'relative', overflow: 'hidden' }}>
      <svg ref={svgRef} width="100%" height="100%" style={{ background: C.bg, display: 'block' }} />
    </div>
  )
}
