'use client'

import { useEffect, useState } from 'react'
import Nav from '@/components/Nav'
import TelemetryChart from '@/components/TelemetryChart'
import DeviceCard from '@/components/DeviceCard'
import HealthBadge from '@/components/HealthBadge'
import { getLiveTelemetry, getTelemetry, getHealth, TelemetryPoint, HealthScore } from '@/lib/api'

const DEVICE_ID = 'tx-001'

export default function Dashboard() {
  const [live, setLive] = useState<TelemetryPoint | null>(null)
  const [history, setHistory] = useState<TelemetryPoint[]>([])
  const [health, setHealth] = useState<HealthScore | null>(null)
  const [scenario, setScenario] = useState<string>('')

  useEffect(() => {
    const fetchLive = async () => {
      try {
        const t = await getLiveTelemetry(DEVICE_ID)
        setLive(t)
        setHistory(prev => [...prev.slice(-59), t])

        const h = await getHealth(DEVICE_ID, t)
        setHealth(h)
      } catch {
        // backend not ready yet
      }
    }

    fetchLive()
    const interval = setInterval(fetchLive, 3000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <Nav />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        <h1 style={{ fontSize: 24, margin: '0 0 24px' }}>Dashboard</h1>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', gap: 16, marginBottom: 24 }}>
          <DeviceCard id={DEVICE_ID} status={live ? 'online' : 'offline'} lastHeartbeat={live?.time} scenario={scenario} />
          {health && <HealthBadge score={health.score} level={health.level} />}
          <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
            <div style={{ fontSize: 12, color: '#64748b', marginBottom: 8 }}>LIVE VALUES</div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div><div style={{ fontSize: 11, color: '#64748b' }}>Temp</div><div style={{ fontSize: 18, fontWeight: 700, color: '#ef4444' }}>{live?.temperature.toFixed(1) ?? '--'}°C</div></div>
              <div><div style={{ fontSize: 11, color: '#64748b' }}>Current</div><div style={{ fontSize: 18, fontWeight: 700, color: '#f59e0b' }}>{live?.current.toFixed(1) ?? '--'}A</div></div>
              <div><div style={{ fontSize: 11, color: '#64748b' }}>Voltage</div><div style={{ fontSize: 18, fontWeight: 700, color: '#38bdf8' }}>{live?.voltage.toFixed(0) ?? '--'}V</div></div>
              <div><div style={{ fontSize: 11, color: '#64748b' }}>Humidity</div><div style={{ fontSize: 18, fontWeight: 700, color: '#a78bfa' }}>{live?.humidity.toFixed(1) ?? '--'}%</div></div>
            </div>
          </div>
        </div>

        <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
          <h2 style={{ fontSize: 16, margin: '0 0 16px' }}>Telemetry (last 60 points)</h2>
          <TelemetryChart data={history} />
        </div>
      </main>
    </div>
  )
}
