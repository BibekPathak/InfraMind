interface Props {
  id: string
  status: string
  lastHeartbeat?: string
  scenario?: string
}

export default function DeviceCard({ id, status, lastHeartbeat, scenario }: Props) {
  const statusColor = status === 'online' ? '#22c55e' : status === 'offline' ? '#ef4444' : '#f59e0b'
  const isLive = lastHeartbeat && Date.now() - new Date(lastHeartbeat).getTime() < 30000

  return (
    <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
        <div style={{ width: 10, height: 10, borderRadius: '50%', background: isLive ? '#22c55e' : statusColor }} />
        <span style={{ fontWeight: 600, fontSize: 14 }}>{id}</span>
      </div>
      <div style={{ fontSize: 12, color: '#94a3b8', display: 'flex', flexDirection: 'column', gap: 4 }}>
        <span>Status: {status}</span>
        {scenario && <span>Scenario: <span style={{ textTransform: 'capitalize', color: '#38bdf8' }}>{scenario.replace('_', ' ')}</span></span>}
        {lastHeartbeat && <span>Last seen: {new Date(lastHeartbeat).toLocaleTimeString()}</span>}
      </div>
    </div>
  )
}
