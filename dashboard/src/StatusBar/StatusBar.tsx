import { useEffect, useState, useRef } from 'react'

type Props = {
  connected: boolean
  reconnecting: boolean
  filesRead: number
  filesWritten: number
  commands: number
  elapsed: number
  status: string
}

function fmt(s: number): string {
  if (s < 1) return '<1s'
  const m = Math.floor(s / 60)
  const sec = Math.floor(s % 60)
  return `${m}:${String(sec).padStart(2, '0')}`
}

const STATUS_COLORS: Record<string, string> = {
  done: '#22c55e',
  idle: '#64748b',
  running: '#fbbf24',
  reading: '#60a5fa',
  writing: '#34d399',
}

const COUNT_COLORS = { reads: '#60a5fa', writes: '#34d399', cmds: '#fbbf24' }

export default function StatusBar({ connected, reconnecting, filesRead, filesWritten, commands, elapsed, status }: Props) {
  const [t, setT] = useState('0:00')
  const start = useRef<number | null>(null)
  const run = status === 'running' || status === 'reading' || status === 'writing'

  useEffect(() => {
    if (run && start.current === null) start.current = Date.now()
    if (status === 'done') { setT(fmt(elapsed)); start.current = null }
    const iv = setInterval(() => { if (start.current !== null) setT(fmt(Math.floor((Date.now() - start.current!) / 1000))) }, 500)
    return () => clearInterval(iv)
  }, [run, status, elapsed])

  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 12,
      padding: '6px 14px', background: '#0d1220',
      borderBottom: '1px solid #1e293b',
      fontSize: 11.5, fontFamily: "'Inter', system-ui, sans-serif",
      color: '#64748b', userSelect: 'none', flexShrink: 0,
    }}>
      {/* Connection */}
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
        <span style={{
          width: 7, height: 7, borderRadius: '50%',
          background: connected ? '#22c55e' : '#ef4444',
          boxShadow: `0 0 7px ${connected ? 'rgba(34,197,94,0.5)' : 'rgba(239,68,68,0.3)'}`,
        }} />
        <span style={{ fontWeight: 650, fontSize: 10.5, letterSpacing: '0.04em', color: connected ? '#22c55e' : '#ef4444' }}>
          {connected ? 'LIVE' : 'OFF'}
        </span>
      </span>

      {reconnecting && <span style={{ color: '#f59e0b', fontStyle: 'italic', fontSize: 10.5 }}>Reconnecting…</span>}

      <span style={{ color: '#1e293b', fontWeight: 100 }}>|</span>

      {/* Status */}
      <span>
        <span style={{ color: STATUS_COLORS[status] || '#e2e8f0', fontWeight: 600 }}>{status}</span>
      </span>

      <span style={{ color: '#1e293b', fontWeight: 100 }}>|</span>

      {/* Counters */}
      <span style={{ display: 'flex', gap: 10 }}>
        <span><span style={{ color: COUNT_COLORS.reads, fontWeight: 600 }}>{filesRead}</span><span style={{ color: '#475569', marginLeft: 2 }}>rd</span></span>
        <span><span style={{ color: COUNT_COLORS.writes, fontWeight: 600 }}>{filesWritten}</span><span style={{ color: '#475569', marginLeft: 2 }}>wr</span></span>
        <span><span style={{ color: COUNT_COLORS.cmds, fontWeight: 600 }}>{commands}</span><span style={{ color: '#475569', marginLeft: 2 }}>cmd</span></span>
      </span>

      <span style={{ flex: 1 }} />

      {/* Elapsed */}
      <span style={{ fontVariantNumeric: 'tabular-nums' }}>
        <span style={{ color: '#475569', fontWeight: 450 }}>⏱</span>{' '}
        <span style={{ color: '#c084fc', fontWeight: 600 }}>{t}</span>
      </span>
    </div>
  )
}
