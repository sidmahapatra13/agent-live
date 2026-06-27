import { useEffect, useState, useRef } from 'react'

type Props = {
  connected: boolean
  filesRead: number
  filesWritten: number
  commands: number
  elapsed: number
  status: string
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

export default function StatusBar({
  connected,
  filesRead,
  filesWritten,
  commands,
  status,
}: Props) {
  const [displayTime, setDisplayTime] = useState('0:00')
  const startRef = useRef<number | null>(null)
  const isRunning = status === 'running' || status === 'reading' || status === 'writing'

  useEffect(() => {
    if (isRunning && startRef.current === null) {
      startRef.current = Date.now()
    }
    if (status === 'done') {
      startRef.current = null
    }

    const interval = setInterval(() => {
      if (startRef.current !== null) {
        const elapsed = Math.floor((Date.now() - startRef.current) / 1000)
        setDisplayTime(formatTime(elapsed))
      }
    }, 500)

    return () => clearInterval(interval)
  }, [isRunning, status])

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 16,
        padding: '8px 16px',
        background: '#0f172a',
        borderBottom: '1px solid #1e293b',
        fontSize: 12,
        fontFamily: "'SF Mono', 'Fira Code', monospace",
        color: '#94a3b8',
        userSelect: 'none',
      }}
    >
      {/* Connection indicator */}
      <span
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6,
        }}
      >
        <span
          style={{
            display: 'inline-block',
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: connected ? '#22c55e' : '#ef4444',
            boxShadow: connected
              ? '0 0 6px rgba(34, 197, 94, 0.5)'
              : '0 0 6px rgba(239, 68, 68, 0.3)',
          }}
        />
        <span style={{ color: connected ? '#22c55e' : '#ef4444', fontWeight: 600 }}>
          {connected ? 'LIVE' : 'OFF'}
        </span>
      </span>

      <span style={{ color: '#334155' }}>|</span>

      {/* Status */}
      <span>
        Status:{' '}
        <span style={{ color: status === 'done' ? '#22c55e' : '#e2e8f0', fontWeight: 600 }}>
          {status}
        </span>
      </span>

      <span style={{ color: '#334155' }}>|</span>

      {/* File counters */}
      <span style={{ display: 'flex', gap: 10 }}>
        <span>
          📖{' '}
          <span style={{ color: '#3b82f6', fontWeight: 600 }}>{filesRead}</span>
        </span>
        <span>
          ✏️{' '}
          <span style={{ color: '#22c55e', fontWeight: 600 }}>{filesWritten}</span>
        </span>
        <span>
          ⚡{' '}
          <span style={{ color: '#eab308', fontWeight: 600 }}>{commands}</span>
        </span>
      </span>

      <span style={{ color: '#334155' }}>|</span>

      {/* Elapsed time */}
      <span>
        ⏱{' '}
        <span style={{ color: '#a78bfa', fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
          {displayTime}
        </span>
      </span>
    </div>
  )
}
