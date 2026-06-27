type Props = {
  connected: boolean
  filesRead: number
  filesWritten: number
  commands: number
  elapsed: number
  status: string
}

export default function StatusBar({
  connected,
  filesRead,
  filesWritten,
  commands,
  status,
}: Props) {
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
      }}
    >
      <span
        style={{
          display: 'inline-block',
          width: 8,
          height: 8,
          borderRadius: '50%',
          background: connected ? '#22c55e' : '#ef4444',
        }}
      />
      <span>{connected ? 'Connected' : 'Disconnected'}</span>
      <span style={{ color: '#475569' }}>|</span>
      <span>
        Status: <strong style={{ color: '#e2e8f0' }}>{status}</strong>
      </span>
      <span style={{ color: '#475569' }}>|</span>
      <span>
        Read:{' '}
        <strong style={{ color: '#3b82f6' }}>{filesRead}</strong>
      </span>
      <span>
        Written:{' '}
        <strong style={{ color: '#22c55e' }}>{filesWritten}</strong>
      </span>
      <span>
        Cmds:{' '}
        <strong style={{ color: '#eab308' }}>{commands}</strong>
      </span>
    </div>
  )
}
