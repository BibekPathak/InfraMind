'use client'

import { useEffect, useState } from 'react'
import Nav from '@/components/Nav'
import TelemetryChart from '@/components/TelemetryChart'
import { getTelemetry, TelemetryPoint } from '@/lib/api'

const DEVICE_ID = 'tx-001'

export default function TelemetryPage() {
  const [data, setData] = useState<TelemetryPoint[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchHistory = async () => {
      try {
        const to = new Date().toISOString()
        const from = new Date(Date.now() - 3600000).toISOString()
        const points = await getTelemetry(DEVICE_ID, from, to)
        setData(points)
      } catch {
        // backend not ready
      }
      setLoading(false)
    }
    fetchHistory()
  }, [])

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <Nav />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        <h1 style={{ fontSize: 24, margin: '0 0 24px' }}>Telemetry History</h1>
        <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
          <p style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 16px' }}>Device: {DEVICE_ID} &mdash; Last 1 hour</p>
          {loading ? (
            <div style={{ height: 400, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#64748b' }}>Loading...</div>
          ) : (
            <TelemetryChart data={data} />
          )}
        </div>
      </main>
    </div>
  )
}
