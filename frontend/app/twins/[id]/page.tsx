'use client'

import { useEffect, useState, useCallback } from 'react'
import Nav from '@/components/Nav'
import TelemetryChart from '@/components/TelemetryChart'
import HealthBadge from '@/components/HealthBadge'
import LiveStream from '@/components/LiveStream'
import { TelemetryPoint } from '@/lib/api'
import { useParams } from 'next/navigation'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

interface TwinEvent {
  id: string
  type: string
  timestamp: string
  summary: string
  details: string
}

interface DigitalTwin {
  assetId: string
  deviceId: string | null
  metadata: Record<string, any>
  liveState: Record<string, any>
  maintenanceHistory: TwinEvent[]
  aiSummary: string
  healthScore: number | null
  healthLevel: string | null
  syncedAt: string | null
  createdAt: string
  updatedAt: string
}

export default function TwinDetailPage() {
  const params = useParams()
  const id = params.id as string

  const [twin, setTwin] = useState<DigitalTwin | null>(null)
  const [telemetry, setTelemetry] = useState<TelemetryPoint[]>([])
  const [loading, setLoading] = useState(true)
  const [newEventType, setNewEventType] = useState('inspection')
  const [newEventSummary, setNewEventSummary] = useState('')
  const [newEventDetails, setNewEventDetails] = useState('')

  const fetchTwin = useCallback(async () => {
    try {
      const res = await fetch(`${API_URL}/api/v1/twins/${id}`)
      if (res.ok) setTwin(await res.json())
    } catch {}
  }, [id])

  useEffect(() => {
    const load = async () => {
      await fetchTwin()
      try {
        const to = new Date().toISOString()
        const from = new Date(Date.now() - 3600000).toISOString()
        const tRes = await fetch(`${API_URL}/api/v1/devices/${id}/telemetry?from=${from}&to=${to}`)
        if (tRes.ok) setTelemetry(await tRes.json())
      } catch {}
      setLoading(false)
    }
    load()
  }, [fetchTwin])

  const handleTwinUpdate = useCallback((data: any) => {
    setTwin(data)
  }, [])

  const handleAddEvent = async () => {
    if (!newEventSummary) return
    try {
      await fetch(`${API_URL}/api/v1/twins/${id}/events`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: newEventType, summary: newEventSummary, details: newEventDetails }),
      })
      setNewEventSummary('')
      setNewEventDetails('')
      await fetchTwin()
    } catch {}
  }

  const health = twin?.healthScore != null ? { score: twin.healthScore, level: twin.healthLevel || 'healthy' } : null

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', background: '#0f172a' }}>
        <Nav />
        <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24, color: '#64748b' }}>Loading...</main>
      </div>
    )
  }

  if (!twin) {
    return (
      <div style={{ minHeight: '100vh', background: '#0f172a' }}>
        <Nav />
        <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24, color: '#64748b' }}>Twin not found.</main>
      </div>
    )
  }

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <LiveStream deviceId={id} onTwinUpdate={handleTwinUpdate} />
      <Nav />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <div>
            <h1 style={{ fontSize: 24, margin: '0 0 4px' }}>Digital Twin</h1>
            <p style={{ fontSize: 14, color: '#94a3b8', margin: 0, fontFamily: 'monospace' }}>{twin.assetId}</p>
          </div>
          {health && <HealthBadge score={health.score} level={health.level} />}
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 24 }}>
          <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
            <h3 style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 12px' }}>LIVE STATE</h3>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, fontSize: 13 }}>
              {Object.entries({
                'Temperature': twin.liveState.temperature,
                'Current': twin.liveState.current,
                'Voltage': twin.liveState.voltage,
                'Humidity': twin.liveState.humidity,
                'Flow Rate': twin.liveState.flowRate,
                'Pressure': twin.liveState.pressure,
                'RPM': twin.liveState.rpm,
                'Vibration': twin.liveState.vibration,
                'Output Power': twin.liveState.outputPower,
                'Device Status': twin.liveState.deviceStatus,
                'Firmware': twin.liveState.firmwareVersion,
                'Devices': twin.liveState.deviceCount,
              }).filter(([, val]) => val !== undefined && val !== null).map(([label, val]) => (
                <div key={label}>
                  <div style={{ color: '#64748b', fontSize: 11 }}>{label}</div>
                  <div style={{ color: '#e2e8f0', fontWeight: 600 }}>{val ?? '--'}</div>
                </div>
              ))}
            </div>
          </div>

          <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
            <h3 style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 12px' }}>METADATA</h3>
            <div style={{ fontSize: 13, color: '#e2e8f0' }}>
              {Object.keys(twin.metadata).length > 0
                ? Object.entries(twin.metadata).map(([k, v]) => (
                    <div key={k} style={{ marginBottom: 4 }}>
                      <span style={{ color: '#64748b' }}>{k}: </span>
                      <span>{String(v)}</span>
                    </div>
                  ))
                : <span style={{ color: '#64748b' }}>No metadata</span>
              }
            </div>
          </div>
        </div>

        {telemetry.length > 0 && (
          <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155', marginBottom: 24 }}>
            <h3 style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 12px' }}>TELEMETRY (LAST HOUR)</h3>
            <TelemetryChart data={telemetry} />
          </div>
        )}

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 24 }}>
          <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
            <h3 style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 12px' }}>MAINTENANCE HISTORY</h3>
            {twin.maintenanceHistory.length === 0 ? (
              <p style={{ color: '#64748b', fontSize: 13 }}>No events recorded.</p>
            ) : (
              <div style={{ maxHeight: 240, overflowY: 'auto' }}>
                {twin.maintenanceHistory.map(evt => (
                  <div key={evt.id} style={{ padding: '8px 0', borderBottom: '1px solid #0f172a', fontSize: 13 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <span style={{ color: '#38bdf8', textTransform: 'capitalize', fontWeight: 600 }}>{evt.type}</span>
                      <span style={{ color: '#64748b', fontSize: 11 }}>{new Date(evt.timestamp).toLocaleString()}</span>
                    </div>
                    <div style={{ color: '#e2e8f0', marginTop: 2 }}>{evt.summary}</div>
                    {evt.details && <div style={{ color: '#94a3b8', fontSize: 12, marginTop: 2 }}>{evt.details}</div>}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
            <h3 style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 12px' }}>ADD EVENT</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <select
                value={newEventType}
                onChange={e => setNewEventType(e.target.value)}
                style={{ padding: '8px 12px', borderRadius: 6, border: '1px solid #334155', background: '#0f172a', color: '#e2e8f0', fontSize: 13 }}
              >
                <option value="inspection">Inspection</option>
                <option value="repair">Repair</option>
                <option value="replacement">Replacement</option>
                <option value="firmware_update">Firmware Update</option>
              </select>
              <input
                placeholder="Summary"
                value={newEventSummary}
                onChange={e => setNewEventSummary(e.target.value)}
                style={{ padding: '8px 12px', borderRadius: 6, border: '1px solid #334155', background: '#0f172a', color: '#e2e8f0', fontSize: 13 }}
              />
              <textarea
                placeholder="Details (optional)"
                value={newEventDetails}
                onChange={e => setNewEventDetails(e.target.value)}
                rows={3}
                style={{ padding: '8px 12px', borderRadius: 6, border: '1px solid #334155', background: '#0f172a', color: '#e2e8f0', fontSize: 13, resize: 'vertical' }}
              />
              <button
                onClick={handleAddEvent}
                disabled={!newEventSummary}
                style={{ padding: '10px', borderRadius: 6, border: 'none', background: newEventSummary ? '#38bdf8' : '#334155', color: newEventSummary ? '#0f172a' : '#64748b', fontSize: 13, fontWeight: 600, cursor: newEventSummary ? 'pointer' : 'not-allowed' }}
              >
                Add Event
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
