'use client'

import { useEffect, useState } from 'react'
import Nav from '@/components/Nav'
import { getAssetTypes, AssetType } from '@/lib/api'

export default function AssetTypesPage() {
  const [types, setTypes] = useState<AssetType[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    getAssetTypes().then(setTypes).catch(() => {}).finally(() => setLoading(false))
  }, [])

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <Nav />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        <h1 style={{ fontSize: 24, margin: '0 0 24px' }}>Asset Types</h1>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>Loading...</div>
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: 16 }}>
            {types.map(t => (
              <div key={t.type} style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                  <div>
                    <div style={{ fontSize: 16, fontWeight: 600, color: '#e2e8f0' }}>{t.displayName}</div>
                    <div style={{ fontSize: 12, color: '#64748b', fontFamily: 'monospace' }}>{t.type}</div>
                  </div>
                  <span style={{
                    padding: '2px 8px', borderRadius: 4, fontSize: 11, fontWeight: 600,
                    background: t.active ? '#22c55e22' : '#ef444422',
                    color: t.active ? '#22c55e' : '#ef4444',
                    textTransform: 'uppercase',
                  }}>
                    {t.active ? 'Active' : 'Inactive'}
                  </span>
                </div>

                <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 8 }}>
                  Metrics: <span style={{ color: '#e2e8f0' }}>{t.metrics.map(m => `${m.name} (${m.unit})`).join(', ')}</span>
                </div>

                <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 8 }}>
                  Health Weights:
                  <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 4 }}>
                    {Object.entries(t.healthWeights).map(([k, v]) => (
                      <span key={k} style={{ padding: '2px 6px', borderRadius: 4, background: '#0f172a', border: '1px solid #334155', color: '#e2e8f0', fontSize: 11 }}>
                        {k}: {v}
                      </span>
                    ))}
                  </div>
                </div>

                <div style={{ fontSize: 12, color: '#94a3b8' }}>
                  Thresholds:
                  <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 4 }}>
                    {Object.entries(t.thresholds).map(([metric, th]) => (
                      <span key={metric} style={{ padding: '2px 6px', borderRadius: 4, background: '#0f172a', border: '1px solid #334155', color: '#94a3b8', fontSize: 11 }}>
                        {metric}: {Object.entries(th).map(([k, v]) => `${k}=${v}`).join(', ')}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
