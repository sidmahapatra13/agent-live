import { useState, useEffect, useRef } from 'react'
import GraphCanvas from './Graph/GraphCanvas'
import Timeline from './Timeline/Timeline'
import StatusBar from './StatusBar/StatusBar'

type Event = {
  type: string
  timestamp: number
  payload: string
  session_id: string
}

export default function App() {
  const [connected, setConnected] = useState(false)
  const [events, setEvents] = useState<Event[]>([])
  const wsRef = useRef<WebSocket | null>(null)

  // TODO Phase 2: derive nodes/edges from events for the graph
  const graphNodes: import('./Graph/GraphCanvas').NodeDef[] = []
  const graphEdges: import('./Graph/GraphCanvas').EdgeDef[] = []

  const stats = {
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
  }

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
        setEvents((prev) => [...prev.slice(-500), event])
      } catch {
        // ignore malformed messages
      }
    }

    return () => ws.close()
  }, [])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <StatusBar connected={connected} {...stats} />
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
        <GraphCanvas nodes={graphNodes} edges={graphEdges} />
        <Timeline events={events} />
      </div>
    </div>
  )
}
