'use client'

import { useEffect, useState } from 'react'
import Nav from '@/components/Nav'
import DeviceCard from '@/components/DeviceCard'
import { getDevices, getAssets, Device, Asset } from '@/lib/api'

export default function DevicesPage() {
  const [assets, setAssets] = useState<Asset[]>([])
  const [devices, setDevices] = useState<Device[]>([])
  const [selectedAsset, setSelectedAsset] = useState<string>('')

  useEffect(() => {
    getAssets().then(setAssets).catch(() => {})
  }, [])

  useEffect(() => {
    if (selectedAsset) {
      getDevices(selectedAsset).then(setDevices).catch(() => {})
    }
  }, [selectedAsset])

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <Nav />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        <h1 style={{ fontSize: 24, margin: '0 0 24px' }}>Devices</h1>

        <div style={{ marginBottom: 24 }}>
          <label style={{ display: 'block', fontSize: 14, color: '#94a3b8', marginBottom: 8 }}>Select Asset</label>
          <select
            value={selectedAsset}
            onChange={e => setSelectedAsset(e.target.value)}
            style={{ padding: '8px 12px', borderRadius: 6, border: '1px solid #334155', background: '#1e293b', color: '#e2e8f0', fontSize: 14, width: 320 }}
          >
            <option value="">-- Select an asset --</option>
            {assets.map(a => (
              <option key={a.id} value={a.id}>{a.name} ({a.type})</option>
            ))}
          </select>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 16 }}>
          {devices.map(d => (
            <DeviceCard key={d.id} id={d.id} status={d.status} lastHeartbeat={d.lastHeartbeat} />
          ))}
          {selectedAsset && devices.length === 0 && (
            <p style={{ color: '#64748b', gridColumn: '1 / -1' }}>No devices registered for this asset.</p>
          )}
        </div>
      </main>
    </div>
  )
}
