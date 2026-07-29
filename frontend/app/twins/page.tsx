'use client'

import { useEffect, useState } from 'react'
import Nav from '@/components/Nav'
import { getAssets, Asset } from '@/lib/api'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

interface DigitalTwin {
  assetId: string
  healthScore: number | null
  healthLevel: string | null
  syncedAt: string | null
  liveState: Record<string, any>
  createdAt: string
}

export default function TwinsPage() {
  const [twins, setTwins] = useState<DigitalTwin[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchTwins = async () => {
      try {
        const res = await fetch(`${API_URL}/api/v1/twins`)
        if (res.ok) {
          const data = await res.json()
          setTwins(data)
        }
      } catch {}
      setLoading(false)
    }
    fetchTwins()
  }, [])

  const healthColor = (level: string | null) =>
    level === 'critical' ? '#ef4444' : level === 'warning' ? '#f59e0b' : level === 'healthy' ? '#22c55e' : '#64748b'

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <Nav />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        <h1 style={{ fontSize: 24, margin: '0 0 24px' }}>Digital Twins</h1>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>Loading...</div>
        ) : twins.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>No twins found. Sync an asset to create its twin.</div>
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 16 }}>
            {twins.map(t => {
              const deviceStatus = t.liveState?.deviceStatus || 'unknown'
              const temp = t.liveState?.temperature
              return (
                <a
                  key={t.assetId}
                  href={`/twins/${t.assetId}`}
                  style={{ textDecoration: 'none' }}
                >
                  <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                      <span style={{ fontWeight: 600, fontSize: 14, color: '#e2e8f0', fontFamily: 'monospace' }}>{t.assetId.slice(0, 8)}</span>
                      <span style={{
                        padding: '2px 8px', borderRadius: 4, fontSize: 11, fontWeight: 600,
                        background: t.healthLevel ? healthColor(t.healthLevel) + '22' : '#64748b22',
                        color: t.healthLevel ? healthColor(t.healthLevel) : '#64748b',
                        textTransform: 'uppercase'
                      }}>
                        {t.healthLevel || 'unknown'}
                      </span>
                    </div>
                    <div style={{ display: 'flex', gap: 16, fontSize: 12, color: '#94a3b8' }}>
                      <div>Status: <span style={{ color: deviceStatus === 'online' ? '#22c55e' : '#ef4444' }}>{deviceStatus}</span></div>
                      {temp != null && <div>Temp: <span style={{ color: temp > 80 ? '#ef4444' : '#e2e8f0' }}>{temp}°C</span></div>}
                    </div>
                    {t.syncedAt && (
                      <div style={{ fontSize: 11, color: '#64748b', marginTop: 8 }}>
                        Synced: {new Date(t.syncedAt).toLocaleTimeString()}
                      </div>
                    )}
                  </div>
                </a>
              )
            })}
          </div>
        )}
      </main>
    </div>
  )
}
