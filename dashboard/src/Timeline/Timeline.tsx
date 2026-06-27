import { useRef, useEffect, useState } from 'react'

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

function describeEvent(ev: Event): string {
  switch (ev.type) {
    case 'file_read':
      return `Read file \`${ev.payload}\``
    case 'file_write':
      return `Wrote to \`${ev.payload}\``
    case 'command':
      return `Ran command: \`${ev.payload}\``
    case 'thought':
      return `Thought: ${ev.payload}`
    case 'plan_step':
      return `Plan step: ${ev.payload}`
    case 'error':
      return `Error: ${ev.payload}`
    case 'done':
      return `Agent finished — ${ev.payload}`
    default:
      return ev.payload
  }
}

export default function Timeline({ events }: Props) {
  const endRef = useRef<HTMLDivElement>(null)
  const [selected, setSelected] = useState<number | null>(null)
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
          const isSelected = selected === i

          return (
            <div key={i}>
              {/* Main event row */}
              <div
                onClick={() => setSelected(isSelected ? null : i)}
                style={{
                  display: 'flex', gap: 7,
                  padding: '5px 10px 5px 10px',
                  fontSize: 12, lineHeight: 1.45,
                  borderLeft: `2px solid ${ev.type === 'done' ? '#22c55e40' : m.color}`,
                  background: isSelected ? 'rgba(148,163,184,0.06)' : 'transparent',
                  cursor: 'pointer',
                  transition: 'background 0.1s',
                  userSelect: 'none',
                }}
                onMouseEnter={e => { if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'rgba(148,163,184,0.04)' }}
                onMouseLeave={e => { if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
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

              {/* Expanded description */}
              {isSelected && (
                <div style={{
                  padding: '6px 10px 8px 34px',
                  fontSize: 11.5,
                  lineHeight: 1.5,
                  color: '#94a3b8',
                  borderLeft: `2px solid ${m.color}44`,
                  background: 'rgba(148,163,184,0.03)',
                }}>
                  {describeEvent(ev)}
                </div>
              )}
            </div>
          )
        })}
      </div>
      <div ref={endRef} />
    </div>
  )
}
