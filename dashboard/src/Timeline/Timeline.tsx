import { useRef, useEffect } from 'react'

type Event = {
  type: string
  timestamp: number
  payload: string
  session_id: string
}

type Props = { events: Event[] }

const EV: Record<string, { color: string; icon: string; label: string }> = {
  file_read:  { color: '#60a5fa', icon: '📖', label: 'read' },
  file_write: { color: '#34d399', icon: '✏️', label: 'write' },
  command:    { color: '#fbbf24', icon: '⚡', label: 'cmd' },
  thought:    { color: '#c084fc', icon: '💭', label: 'think' },
  plan_step:  { color: '#22d3ee', icon: '🎯', label: 'plan' },
  error:      { color: '#ef4444', icon: '❌', label: 'error' },
  done:       { color: '#22c55e', icon: '✅', label: 'done' },
}

export default function Timeline({ events }: Props) {
  const endRef = useRef<HTMLDivElement>(null)
  useEffect(() => { endRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [events.length])

  if (events.length === 0) {
    return (
      <div style={{
        width: 300, background: '#0d1220',
        borderLeft: '1px solid #1e293b',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        flexDirection: 'column', gap: 8,
        color: '#475569', fontSize: 13,
        fontFamily: "'Inter', system-ui, sans-serif",
      }}>
        <span style={{ fontSize: 22, opacity: 0.35 }}>⚡</span>
        <span>Waiting for agent events…</span>
      </div>
    )
  }

  return (
    <div style={{
      width: 300, minWidth: 260, background: '#0d1220',
      borderLeft: '1px solid #1e293b',
      overflowY: 'auto', overflowX: 'hidden',
      fontFamily: "'Inter', system-ui, sans-serif",
    }}>
      {/* Header */}
      <div style={{
        padding: '7px 12px', borderBottom: '1px solid #1e293b',
        fontSize: 10, fontWeight: 600, color: '#475569',
        letterSpacing: '0.06em', textTransform: 'uppercase',
        display: 'flex', justifyContent: 'space-between',
        position: 'sticky', top: 0, background: '#0d1220', zIndex: 1,
      }}>
        <span>Events</span>
        <span style={{ color: '#64748b' }}>{events.length}</span>
      </div>

      {/* List */}
      <div style={{ padding: '3px 0' }}>
        {events.map((ev, i) => {
          const m = EV[ev.type]
          if (!m) return null

          return (
            <div key={i} style={{
              display: 'flex', gap: 7,
              padding: '5px 10px 5px 10px',
              fontSize: 12, lineHeight: 1.45,
              borderLeft: `2px solid ${ev.type === 'done' ? '#22c55e40' : m.color}`,
              transition: 'background 0.1s',
            }}
             onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = 'rgba(148,163,184,0.04)'}
             onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = 'transparent'}
            >
              {/* Icon column */}
              <span style={{ flexShrink: 0, fontSize: 12.5, lineHeight: '17px', width: 16, textAlign: 'center' }}>
                {m.icon}
              </span>

              {/* Body */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{
                  color: ev.type === 'done' ? '#22c55e' : '#e2e8f0',
                  fontWeight: ev.type === 'done' ? 500 : 400,
                  overflowWrap: 'break-word',
                  wordBreak: 'break-word',
                }}>
                  {ev.payload}
                </div>
                <div style={{
                  display: 'flex', gap: 5, fontSize: 10,
                  color: '#475569', marginTop: 1,
                }}>
                  <span>{m.label}</span>
                  <span>·</span>
                  <span style={{ fontVariantNumeric: 'tabular-nums' }}>{ev.timestamp.toFixed(1)}s</span>
                </div>
              </div>
            </div>
          )
        })}
      </div>
      <div ref={endRef} />
    </div>
  )
}
