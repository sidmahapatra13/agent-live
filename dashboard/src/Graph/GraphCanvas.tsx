import { useEffect, useRef } from 'react'

export type NodeDef = {
  id: string
  label: string
  kind: string
  event_type: string
}

export type EdgeDef = {
  source: string
  target: string
  kind: string
}

type Props = {
  nodes: NodeDef[]
  edges: EdgeDef[]
}

export default function GraphCanvas({ nodes, edges }: Props) {
  const svgRef = useRef<SVGSVGElement>(null)

  // TODO Phase 2: D3-force simulation with nodes, edges, agent particle
  useEffect(() => {
    if (!svgRef.current) return
  }, [nodes, edges])

  return (
    <svg
      ref={svgRef}
      width="100%"
      height="100%"
      style={{ background: '#0a0e17', flex: 1 }}
    />
  )
}
