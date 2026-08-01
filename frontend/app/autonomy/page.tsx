'use client'

import { useEffect, useState, useCallback } from 'react'
import Nav from '@/components/Nav'
import { getAssets, Asset } from '@/lib/api'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

interface Action {
  id: string
  assetId: string
  deviceId: string | null
  type: string
  payload: Record<string, any>
  source: string
  status: string
  approvalRequired: boolean
  autoApproved: boolean
  reason: string
  result: string | null
  proposedAt: string
  executedAt: string | null
}

const STATUS_COLORS: Record<string, string> = {
  proposed: '#3b82f6',
  approved: '#f59e0b',
  rejected: '#ef4444',
  executed: '#22c55e',
  failed: '#ef4444',
}

const TYPE_COLORS: Record<string, string> = {
  command: '#38bdf8',
  restart: '#f97316',
  config_change: '#a78bfa',
  notification: '#22c55e',
}

export default function AutonomyPage() {
  const [actions, setActions] = useState<Action[]>([])
  const [assets, setAssets] = useState<Asset[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('')

  const fetchActions = useCallback(async () => {
    try {
      const params = new URLSearchParams()
      if (statusFilter) params.set('status', statusFilter)
      const res = await fetch(`${API_URL}/api/v1/actions?${params}`)
      if (res.ok) setActions(await res.json())
    } catch {}
    setLoading(false)
  }, [statusFilter])

  useEffect(() => {
    fetchActions()
  }, [fetchActions])

  useEffect(() => {
    getAssets().then(setAssets).catch(() => {})
  }, [])

  const handleApprove = async (id: string) => {
    try {
      await fetch(`${API_URL}/api/v1/actions/${id}/approve`, { method: 'PATCH' })
      fetchActions()
    } catch {}
  }

  const handleReject = async (id: string) => {
    try {
      await fetch(`${API_URL}/api/v1/actions/${id}/reject`, { method: 'PATCH' })
      fetchActions()
    } catch {}
  }

  const handleAutonomyChange = async (assetId: string, mode: string) => {
    try {
      await fetch(`${API_URL}/api/v1/assets/${assetId}/autonomy`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ autonomyMode: mode }),
      })
      const updated = await getAssets()
      setAssets(updated)
    } catch {}
  }

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <Nav />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <h1 style={{ fontSize: 24, margin: 0 }}>Autonomy</h1>
          <select
            value={statusFilter}
            onChange={e => setStatusFilter(e.target.value)}
            style={{ padding: '8px 12px', borderRadius: 6, border: '1px solid #334155', background: '#1e293b', color: '#e2e8f0', fontSize: 14 }}
          >
            <option value="">All Status</option>
            {Object.keys(STATUS_COLORS).map(s => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>

        <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155', marginBottom: 24 }}>
          <h3 style={{ fontSize: 13, color: '#94a3b8', margin: '0 0 12px' }}>ASSET AUTONOMY MODES</h3>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
            {assets.map(a => (
              <div key={a.id} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', borderRadius: 6, background: '#0f172a', border: '1px solid #334155' }}>
                <span style={{ fontSize: 13, color: '#e2e8f0' }}>{a.name}</span>
                <select
                  value={a.autonomyMode || 'manual'}
                  onChange={e => handleAutonomyChange(a.id, e.target.value)}
                  style={{ padding: '4px 8px', borderRadius: 4, border: '1px solid #334155', background: '#1e293b', color: '#e2e8f0', fontSize: 12 }}
                >
                  <option value="manual">Manual</option>
                  <option value="advisory">Advisory</option>
                  <option value="autonomous">Autonomous</option>
                </select>
              </div>
            ))}
          </div>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>Loading...</div>
        ) : actions.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>No actions found.</div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {actions.map(a => (
              <div key={a.id} style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{
                      padding: '2px 8px', borderRadius: 4, fontSize: 11, fontWeight: 600,
                      background: (TYPE_COLORS[a.type] || '#64748b') + '22',
                      color: TYPE_COLORS[a.type] || '#64748b',
                      textTransform: 'uppercase',
                    }}>
                      {a.type.replace('_', ' ')}
                    </span>
                    <span style={{
                      padding: '2px 8px', borderRadius: 4, fontSize: 11, fontWeight: 600,
                      background: (STATUS_COLORS[a.status] || '#64748b') + '22',
                      color: STATUS_COLORS[a.status] || '#64748b',
                      textTransform: 'uppercase',
                    }}>
                      {a.status}
                    </span>
                    {a.autoApproved && <span style={{ fontSize: 11, color: '#22c55e' }}>AUTO</span>}
                    {a.approvalRequired && a.status === 'proposed' && <span style={{ fontSize: 11, color: '#f59e0b' }}>REQUIRES APPROVAL</span>}
                  </div>
                  <span style={{ fontSize: 12, color: '#64748b' }}>source: {a.source}</span>
                </div>

                {a.reason && <div style={{ fontSize: 13, color: '#cbd5e1', marginBottom: 4 }}>{a.reason}</div>}
                {Object.keys(a.payload).length > 0 && (
                  <div style={{ fontSize: 12, color: '#64748b', fontFamily: 'monospace', marginBottom: 8 }}>
                    {JSON.stringify(a.payload)}
                  </div>
                )}
                <div style={{ fontSize: 11, color: '#64748b', marginBottom: 12 }}>
                  Proposed: {new Date(a.proposedAt).toLocaleString()}
                  {a.executedAt && <> · Executed: {new Date(a.executedAt).toLocaleString()}</>}
                  {a.result && <> · {a.result}</>}
                </div>

                {a.status === 'proposed' && (
                  <div style={{ display: 'flex', gap: 8 }}>
                    <button onClick={() => handleApprove(a.id)}
                      style={{ padding: '6px 14px', borderRadius: 6, border: 'none', background: '#22c55e', color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
                      Approve
                    </button>
                    <button onClick={() => handleReject(a.id)}
                      style={{ padding: '6px 14px', borderRadius: 6, border: '1px solid #334155', background: 'transparent', color: '#ef4444', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
                      Reject
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
