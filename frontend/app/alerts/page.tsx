'use client'

import { useEffect, useState, useCallback } from 'react'
import Nav from '@/components/Nav'
import LiveStream from '@/components/LiveStream'
import { TelemetryPoint } from '@/lib/api'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

interface Alert {
  id: string
  deviceId: string
  severity: string
  rule: string
  message: string
  status: string
  createdAt: string
  updatedAt: string
}

interface WSEvent {
  type: string
  timestamp: string
  payload: Alert
}

export default function AlertsPage() {
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [filter, setFilter] = useState('')
  const [loading, setLoading] = useState(true)

  const fetchAlerts = useCallback(async () => {
    try {
      const params = new URLSearchParams()
      if (filter) params.set('status', filter)
      params.set('limit', '50')
      const res = await fetch(`${API_URL}/api/v1/alerts?${params}`)
      if (res.ok) {
        const data = await res.json()
        setAlerts(data)
      }
    } catch {
      // backend not ready
    }
    setLoading(false)
  }, [filter])

  useEffect(() => {
    fetchAlerts()
  }, [fetchAlerts])

  const handleWSEvent = useCallback((point: TelemetryPoint) => {
    // WebSocket only handles telemetry; alerts fetched via REST
  }, [])

  const handleAcknowledge = async (id: string) => {
    try {
      const res = await fetch(`${API_URL}/api/v1/alerts/${id}/acknowledge`, { method: 'PATCH' })
      if (res.ok) fetchAlerts()
    } catch {}
  }

  const handleResolve = async (id: string) => {
    try {
      const res = await fetch(`${API_URL}/api/v1/alerts/${id}/resolve`, { method: 'PATCH' })
      if (res.ok) fetchAlerts()
    } catch {}
  }

  const severityColor = (s: string) => s === 'critical' ? '#ef4444' : s === 'warning' ? '#f59e0b' : '#3b82f6'
  const statusColor = (s: string) => s === 'open' ? '#ef4444' : s === 'acknowledged' ? '#f59e0b' : '#22c55e'

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <LiveStream deviceId="tx-001" onTelemetry={handleWSEvent} />
      <Nav />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <h1 style={{ fontSize: 24, margin: 0 }}>Alerts</h1>
          <select
            value={filter}
            onChange={e => setFilter(e.target.value)}
            style={{ padding: '8px 12px', borderRadius: 6, border: '1px solid #334155', background: '#1e293b', color: '#e2e8f0', fontSize: 14 }}
          >
            <option value="">All Status</option>
            <option value="open">Open</option>
            <option value="acknowledged">Acknowledged</option>
            <option value="resolved">Resolved</option>
          </select>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>Loading...</div>
        ) : alerts.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>No alerts found.</div>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #334155', color: '#94a3b8', textAlign: 'left' }}>
                  <th style={{ padding: '12px 8px' }}>Severity</th>
                  <th style={{ padding: '12px 8px' }}>Device</th>
                  <th style={{ padding: '12px 8px' }}>Rule</th>
                  <th style={{ padding: '12px 8px' }}>Message</th>
                  <th style={{ padding: '12px 8px' }}>Status</th>
                  <th style={{ padding: '12px 8px' }}>Time</th>
                  <th style={{ padding: '12px 8px' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {alerts.map(a => (
                  <tr key={a.id} style={{ borderBottom: '1px solid #1e293b' }}>
                    <td style={{ padding: '10px 8px' }}>
                      <span style={{ color: severityColor(a.severity), fontWeight: 600, textTransform: 'uppercase', fontSize: 12 }}>
                        {a.severity}
                      </span>
                    </td>
                    <td style={{ padding: '10px 8px', color: '#94a3b8', fontFamily: 'monospace', fontSize: 12 }}>{a.deviceId}</td>
                    <td style={{ padding: '10px 8px', color: '#e2e8f0' }}>{a.rule}</td>
                    <td style={{ padding: '10px 8px', color: '#cbd5e1' }}>{a.message}</td>
                    <td style={{ padding: '10px 8px' }}>
                      <span style={{ color: statusColor(a.status), fontWeight: 600, textTransform: 'uppercase', fontSize: 12 }}>
                        {a.status}
                      </span>
                    </td>
                    <td style={{ padding: '10px 8px', color: '#64748b', fontSize: 12 }}>
                      {new Date(a.createdAt).toLocaleString()}
                    </td>
                    <td style={{ padding: '10px 8px' }}>
                      {a.status === 'open' && (
                        <button onClick={() => handleAcknowledge(a.id)}
                          style={{ padding: '4px 10px', borderRadius: 4, border: 'none', background: '#3b82f6', color: '#fff', fontSize: 12, cursor: 'pointer', marginRight: 4 }}>
                          Acknowledge
                        </button>
                      )}
                      {a.status !== 'resolved' && (
                        <button onClick={() => handleResolve(a.id)}
                          style={{ padding: '4px 10px', borderRadius: 4, border: 'none', background: '#22c55e', color: '#fff', fontSize: 12, cursor: 'pointer' }}>
                          Resolve
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  )
}
