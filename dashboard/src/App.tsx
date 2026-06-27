import { useState, useEffect, useRef, useMemo } from 'react'
import GraphCanvas, { NodeDef, EdgeDef } from './Graph/GraphCanvas'
import Timeline from './Timeline/Timeline'
import StatusBar from './StatusBar/StatusBar'

type Event = {
  type: string
  timestamp: number
  payload: string
  session_id: string
}

// Derive a unique node ID from a file path or command string
function nodeId(kind: string, payload: string): string {
  return `${kind}::${payload}`
}

// Extract a short label from a path or command
function shortLabel(kind: string, payload: string): string {
  if (kind === 'file') {
    const parts = payload.split('/')
    return parts[parts.length - 1] || payload
  }
  if (payload.length > 40) return payload.slice(0, 37) + '...'
  return payload
}

export default function App() {
  const [connected, setConnected] = useState(false)
  const [events, setEvents] = useState<Event[]>([])
  const wsRef = useRef<WebSocket | null>(null)

  // Track graph state via refs (avoids re-render cascades)
  const nodeMapRef = useRef<Map<string, NodeDef>>(new Map())
  const edgeListRef = useRef<EdgeDef[]>([])
  const lastNodeRef = useRef<string | null>(null)

  // Force re-render trigger for graph (incremented when data changes)
  const [graphTick, setGraphTick] = useState(0)

  // Derived stats from events
  const stats = useMemo(() => ({
    filesRead: events.filter((e) => e.type === 'file_read').length,
    filesWritten: events.filter((e) => e.type === 'file_write').length,
    commands: events.filter((e) => e.type === 'command').length,
    elapsed: events.length > 0 ? events[events.length - 1].timestamp : 0,
    status:
      events.length === 0
        ? 'idle'
        : events[events.length - 1].type === 'done'
          ? 'done'
          : 'running',
  }), [events])

  // Refs so the event handler always reads fresh data
  const nodeMap = nodeMapRef
  const edgeList = edgeListRef
  const lastNode = lastNodeRef

  // Process an event into graph nodes/edges
  const processEvent = useRef((event: Event) => {
    const { type, payload } = event
    const nm = nodeMap.current
    const el = edgeList.current
    const AGENT = '__agent__'

    switch (type) {
      case 'file_read': {
        const id = nodeId('file', payload)
        if (!nm.has(id)) {
          nm.set(id, { id, label: shortLabel('file', payload), kind: 'file', event_type: 'file_read' })
        }
        // Edge from agent to file
        el.push({ source: AGENT, target: id, kind: 'read' })
        lastNode.current = id
        break
      }
      case 'file_write': {
        const id = nodeId('file', payload)
        if (!nm.has(id)) {
          nm.set(id, { id, label: shortLabel('file', payload), kind: 'file', event_type: 'file_write' })
        } else {
          // Update existing node's event type
          const existing = nm.get(id)!
          nm.set(id, { ...existing, event_type: 'file_write' })
        }
        el.push({ source: AGENT, target: id, kind: 'write' })
        lastNode.current = id
        break
      }
      case 'command': {
        const id = nodeId('command', payload)
        if (!nm.has(id)) {
          nm.set(id, { id, label: shortLabel('command', payload), kind: 'command', event_type: 'command' })
        }
        el.push({ source: AGENT, target: id, kind: 'exec' })
        // Connect command to the last file node if it exists
        if (lastNode.current && lastNode.current !== id) {
          el.push({ source: id, target: lastNode.current, kind: 'exec' })
        }
        lastNode.current = id
        break
      }
      case 'thought': {
        if (payload.length < 15) break // skip short thoughts
        const id = nodeId('thought', payload)
        if (!nm.has(id)) {
          nm.set(id, { id, label: shortLabel('thought', payload), kind: 'thought', event_type: 'thought' })
        }
        // Connect thought to last active node
        if (lastNode.current) {
          el.push({ source: lastNode.current, target: id, kind: 'read' })
        }
        lastNode.current = id
        break
      }
      case 'plan_step': {
        const id = nodeId('thought', `plan:${payload}`)
        if (!nm.has(id)) {
          nm.set(id, { id, label: shortLabel('thought', `🎯 ${payload}`), kind: 'thought', event_type: 'plan_step' })
        }
        lastNode.current = id
        break
      }
    }

    // Trim edge list to keep perf reasonable
    if (el.length > 1000) {
      edgeList.current = el.slice(-800)
    }

    setGraphTick((t) => t + 1)
  })

  // ── WebSocket connection ────────────────────────────
  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => setConnected(true)
    ws.onclose = () => setConnected(false)
    ws.onerror = () => setConnected(false)

    ws.onmessage = (msg) => {
      try {
        const event = JSON.parse(msg.data) as Event
        console.log('[App] Event:', event.type, event.payload.slice(0, 60))
        setEvents((prev) => [...prev.slice(-500), event])
        processEvent.current(event)
      } catch {
        // ignore malformed
      }
    }

    return () => ws.close()
  }, [])

  // Memoize graph data for the canvas
  const graphData = useMemo(() => ({
    nodes: Array.from(nodeMapRef.current.values()),
    edges: edgeListRef.current,
    agentPosition: lastNodeRef.current
      ? { source: '__agent__', target: lastNodeRef.current }
      : null,
  }), [graphTick])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <StatusBar connected={connected} {...stats} />
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        <GraphCanvas
          nodes={graphData.nodes}
          edges={graphData.edges}
          agentPosition={graphData.agentPosition}
        />
        <Timeline events={events} />
      </div>
    </div>
  )
}
