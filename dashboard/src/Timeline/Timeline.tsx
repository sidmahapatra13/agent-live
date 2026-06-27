import { useRef, useEffect } from 'react'

type Event = {
  type: string
  timestamp: number
  payload: string
  session_id: string
}

type Props = {
  events: Event[]
}

const eventColors: Record<string, string> = {
  file_read: '#3b82f6',
  file_write: '#22c55e',
  command: '#eab308',
  thought: '#a855f7',
  plan_step: '#06b6d4',
  error: '#ef4444',
  done: '#22c55e',
}

const eventIcons: Record<string, string> = {
  file_read: '📖',
  file_write: '✏️',
  command: '⚡',
  thought: '💭',
  plan_step: '🎯',
  error: '❌',
  done: '✅',
}

export default function Timeline({ events }: Props) {
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [events.length])

  return (
    <div
      style={{
        width: 320,
        background: '#111827',
        borderLeft: '1px solid #1f2937',
        overflowY: 'auto',
        padding: '8px 0',
      }}
    >
      {events.length === 0 && (
        <div style={{ padding: 16, color: '#6b7280', fontSize: 13 }}>
          Waiting for agent events...
        </div>
      )}
      {events.map((ev, i) => (
        <div
          key={i}
          style={{
            padding: '4px 12px',
            borderLeft: `3px solid ${eventColors[ev.type] || '#6b7280'}`,
            margin: '2px 0',
            fontSize: 12,
            fontFamily: "'SF Mono', 'Fira Code', monospace",
          }}
        >
          <span style={{ marginRight: 4 }}>{eventIcons[ev.type] || '•'}</span>
          <span style={{ color: '#9ca3af', fontSize: 11 }}>
            {ev.timestamp.toFixed(1)}s
          </span>{' '}
          <span style={{ color: '#d1d5db' }}>{ev.payload}</span>
        </div>
      ))}
      <div ref={endRef} />
    </div>
  )
}
