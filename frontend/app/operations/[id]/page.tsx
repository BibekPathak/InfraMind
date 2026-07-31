'use client'

import { useEffect, useState } from 'react'
import Nav from '@/components/Nav'
import { useParams } from 'next/navigation'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

interface TimelineEvent {
  timestamp: string
  action: string
  actor: string
  note: string
}

interface WorkOrder {
  id: string
  assetId: string
  alertId: string | null
  type: string
  priority: string
  status: string
  assignedTo: string | null
  estimatedCost: number | null
  description: string
  timeline: TimelineEvent[]
  createdAt: string
  updatedAt: string
}

export default function WorkOrderDetailPage() {
  const params = useParams()
  const id = params.id as string

  const [order, setOrder] = useState<WorkOrder | null>(null)
  const [loading, setLoading] = useState(true)
  const [assignee, setAssignee] = useState('')

  const fetchOrder = async () => {
    try {
      const res = await fetch(`${API_URL}/api/v1/work-orders/${id}`)
      if (res.ok) setOrder(await res.json())
    } catch {}
  }

  useEffect(() => {
    fetchOrder()
  }, [id])

  const handleAssign = async () => {
    if (!assignee) return
    try {
      await fetch(`${API_URL}/api/v1/work-orders/${id}/assign`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ assignedTo: assignee }),
      })
      setAssignee('')
      fetchOrder()
    } catch {}
  }

  const handleStatus = async (status: string) => {
    try {
      await fetch(`${API_URL}/api/v1/work-orders/${id}/status`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status }),
      })
      fetchOrder()
    } catch {}
  }

  const priorityColor = (p: string) => p === 'critical' ? '#ef4444' : p === 'high' ? '#f59e0b' : p === 'medium' ? '#3b82f6' : '#64748b'
  const statusColor = (s: string) => {
    const map: Record<string, string> = {
      open: '#3b82f6', assigned: '#f59e0b', in_progress: '#a78bfa',
      completed: '#22c55e', cancelled: '#64748b',
    }
    return map[s] || '#64748b'
  }

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', background: '#0f172a' }}>
        <Nav />
        <main style={{ maxWidth: 1000, margin: '0 auto', padding: 24, color: '#64748b' }}>Loading...</main>
      </div>
    )
  }

  if (!order) {
    return (
      <div style={{ minHeight: '100vh', background: '#0f172a' }}>
        <Nav />
        <main style={{ maxWidth: 1000, margin: '0 auto', padding: 24, color: '#64748b' }}>Work order not found.</main>
      </div>
    )
  }

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <Nav />
      <main style={{ maxWidth: 1000, margin: '0 auto', padding: 24 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 24 }}>
          <div>
            <h1 style={{ fontSize: 24, margin: '0 0 4px' }}>Work Order</h1>
            <p style={{ fontSize: 14, color: '#64748b', margin: 0, fontFamily: 'monospace' }}>{order.id}</p>
          </div>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <span style={{ padding: '4px 12px', borderRadius: 4, fontSize: 12, fontWeight: 600, background: priorityColor(order.priority) + '22', color: priorityColor(order.priority), textTransform: 'uppercase' }}>
              {order.priority}
            </span>
            <span style={{ padding: '4px 12px', borderRadius: 4, fontSize: 12, fontWeight: 600, background: statusColor(order.status) + '22', color: statusColor(order.status), textTransform: 'uppercase' }}>
              {order.status.replace('_', ' ')}
            </span>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 24 }}>
          <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
            <h3 style={{ fontSize: 13, color: '#94a3b8', margin: '0 0 12px' }}>DETAILS</h3>
            <div style={{ fontSize: 13, display: 'flex', flexDirection: 'column', gap: 6 }}>
              <div><span style={{ color: '#64748b' }}>Type: </span><span style={{ color: '#e2e8f0', textTransform: 'capitalize' }}>{order.type.replace('_', ' ')}</span></div>
              <div><span style={{ color: '#64748b' }}>Asset: </span><span style={{ color: '#e2e8f0', fontFamily: 'monospace' }}>{order.assetId}</span></div>
              <div><span style={{ color: '#64748b' }}>Assigned: </span><span style={{ color: '#e2e8f0' }}>{order.assignedTo || 'Unassigned'}</span></div>
              {order.estimatedCost != null && <div><span style={{ color: '#64748b' }}>Est. Cost: </span><span style={{ color: '#e2e8f0' }}>${order.estimatedCost.toLocaleString()}</span></div>}
              {order.description && <div><span style={{ color: '#64748b' }}>Description: </span><span style={{ color: '#e2e8f0' }}>{order.description}</span></div>}
            </div>
          </div>

          <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
            <h3 style={{ fontSize: 13, color: '#94a3b8', margin: '0 0 12px' }}>ACTIONS</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {!order.assignedTo && (
                <div style={{ display: 'flex', gap: 8 }}>
                  <input
                    placeholder="Engineer name"
                    value={assignee}
                    onChange={e => setAssignee(e.target.value)}
                    style={{ flex: 1, padding: '8px 12px', borderRadius: 6, border: '1px solid #334155', background: '#0f172a', color: '#e2e8f0', fontSize: 13 }}
                  />
                  <button onClick={handleAssign} disabled={!assignee}
                    style={{ padding: '8px 16px', borderRadius: 6, border: 'none', background: assignee ? '#3b82f6' : '#334155', color: assignee ? '#fff' : '#64748b', fontSize: 13, cursor: assignee ? 'pointer' : 'not-allowed' }}>
                    Assign
                  </button>
                </div>
              )}
              <div style={{ display: 'flex', gap: 8 }}>
                {order.status === 'assigned' && (
                  <button onClick={() => handleStatus('in_progress')}
                    style={{ flex: 1, padding: '10px', borderRadius: 6, border: 'none', background: '#a78bfa', color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
                    Start Work
                  </button>
                )}
                {order.status === 'in_progress' && (
                  <button onClick={() => handleStatus('completed')}
                    style={{ flex: 1, padding: '10px', borderRadius: 6, border: 'none', background: '#22c55e', color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
                    Complete
                  </button>
                )}
                {(order.status === 'open' || order.status === 'assigned' || order.status === 'in_progress') && (
                  <button onClick={() => handleStatus('cancelled')}
                    style={{ flex: 1, padding: '10px', borderRadius: 6, border: '1px solid #334155', background: 'transparent', color: '#94a3b8', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
                    Cancel
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>

        <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
          <h3 style={{ fontSize: 13, color: '#94a3b8', margin: '0 0 16px' }}>INCIDENT TIMELINE</h3>
          <div style={{ position: 'relative', paddingLeft: 24 }}>
            <div style={{ position: 'absolute', left: 8, top: 4, bottom: 4, width: 2, background: '#334155' }} />
            {order.timeline.map((evt, i) => (
              <div key={i} style={{ position: 'relative', paddingBottom: 16 }}>
                <div style={{ position: 'absolute', left: -20, top: 4, width: 8, height: 8, borderRadius: '50%', background: '#38bdf8' }} />
                <div style={{ fontSize: 13, color: '#e2e8f0' }}>
                  <span style={{ color: '#38bdf8', fontWeight: 600, textTransform: 'capitalize' }}>{evt.action.replace('_', ' ')}</span>
                  <span style={{ color: '#64748b', fontSize: 11, marginLeft: 8 }}>{new Date(evt.timestamp).toLocaleString()}</span>
                </div>
                {evt.note && <div style={{ fontSize: 12, color: '#94a3b8', marginTop: 2 }}>{evt.note}</div>}
              </div>
            ))}
          </div>
        </div>
      </main>
    </div>
  )
}
