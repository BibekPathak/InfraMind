'use client'

import { useEffect, useState, useCallback } from 'react'
import Nav from '@/components/Nav'

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

const STATUS_ORDER = ['open', 'assigned', 'in_progress', 'completed', 'cancelled']

export default function OperationsPage() {
  const [orders, setOrders] = useState<WorkOrder[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('')
  const [priorityFilter, setPriorityFilter] = useState('')
  const [assignModal, setAssignModal] = useState<WorkOrder | null>(null)
  const [assignee, setAssignee] = useState('')

  const fetchOrders = useCallback(async () => {
    try {
      const params = new URLSearchParams()
      if (statusFilter) params.set('status', statusFilter)
      if (priorityFilter) params.set('priority', priorityFilter)
      const res = await fetch(`${API_URL}/api/v1/work-orders?${params}`)
      if (res.ok) setOrders(await res.json())
    } catch {}
    setLoading(false)
  }, [statusFilter, priorityFilter])

  useEffect(() => {
    fetchOrders()
  }, [fetchOrders])

  const handleAssign = async () => {
    if (!assignModal || !assignee) return
    try {
      await fetch(`${API_URL}/api/v1/work-orders/${assignModal.id}/assign`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ assignedTo: assignee }),
      })
      setAssignee('')
      setAssignModal(null)
      fetchOrders()
    } catch {}
  }

  const handleStatus = async (order: WorkOrder, status: string) => {
    try {
      await fetch(`${API_URL}/api/v1/work-orders/${order.id}/status`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status }),
      })
      fetchOrders()
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

  const grouped = STATUS_ORDER.map(status => ({
    status,
    items: orders.filter(o => o.status === status),
  }))

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <Nav />
      <main style={{ maxWidth: 1400, margin: '0 auto', padding: 24 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <h1 style={{ fontSize: 24, margin: 0 }}>Operations</h1>
          <div style={{ display: 'flex', gap: 8 }}>
            <select value={statusFilter} onChange={e => setStatusFilter(e.target.value)}
              style={{ padding: '8px 12px', borderRadius: 6, border: '1px solid #334155', background: '#1e293b', color: '#e2e8f0', fontSize: 14 }}>
              <option value="">All Status</option>
              {STATUS_ORDER.map(s => <option key={s} value={s}>{s.replace('_', ' ')}</option>)}
            </select>
            <select value={priorityFilter} onChange={e => setPriorityFilter(e.target.value)}
              style={{ padding: '8px 12px', borderRadius: 6, border: '1px solid #334155', background: '#1e293b', color: '#e2e8f0', fontSize: 14 }}>
              <option value="">All Priority</option>
              {['low', 'medium', 'high', 'critical'].map(p => <option key={p} value={p}>{p}</option>)}
            </select>
          </div>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>Loading...</div>
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 16 }}>
            {grouped.map(group => (
              <div key={group.status}>
                <div style={{
                  fontSize: 13, fontWeight: 600, textTransform: 'uppercase', letterSpacing: 1,
                  color: statusColor(group.status), marginBottom: 12,
                }}>
                  {group.status.replace('_', ' ')} ({group.items.length})
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                  {group.items.length === 0 ? (
                    <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px dashed #334155', color: '#475569', fontSize: 13, textAlign: 'center' }}>
                      Empty
                    </div>
                  ) : group.items.map(wo => (
                    <div key={wo.id} style={{ background: '#1e293b', borderRadius: 8, padding: 14, border: '1px solid #334155' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                        <span style={{ fontSize: 12, color: priorityColor(wo.priority), fontWeight: 700, textTransform: 'uppercase' }}>{wo.priority}</span>
                        <span style={{ fontSize: 11, color: '#64748b', textTransform: 'capitalize' }}>{wo.type.replace('_', ' ')}</span>
                      </div>
                      <a href={`/operations/${wo.id}`} style={{ fontSize: 14, fontWeight: 600, color: '#e2e8f0', textDecoration: 'none', display: 'block', marginBottom: 4 }}>
                        {wo.description || wo.type}
                      </a>
                      <div style={{ fontSize: 11, color: '#64748b', marginBottom: 4 }}>
                        Asset: <span style={{ fontFamily: 'monospace' }}>{wo.assetId.slice(0, 8)}</span>
                        {wo.assignedTo && <> · Assigned: <span style={{ color: '#94a3b8' }}>{wo.assignedTo}</span></>}
                      </div>
                      <div style={{ fontSize: 11, color: '#64748b', marginBottom: 10 }}>
                        {new Date(wo.createdAt).toLocaleString()}
                      </div>
                      <div style={{ display: 'flex', gap: 6 }}>
                        {!wo.assignedTo && (
                          <button onClick={() => setAssignModal(wo)}
                            style={{ padding: '4px 10px', borderRadius: 4, border: 'none', background: '#3b82f6', color: '#fff', fontSize: 12, cursor: 'pointer' }}>
                            Assign
                          </button>
                        )}
                        {wo.status === 'open' && wo.assignedTo && (
                          <button onClick={() => handleStatus(wo, 'in_progress')}
                            style={{ padding: '4px 10px', borderRadius: 4, border: 'none', background: '#a78bfa', color: '#fff', fontSize: 12, cursor: 'pointer' }}>
                            Start
                          </button>
                        )}
                        {wo.status === 'in_progress' && (
                          <button onClick={() => handleStatus(wo, 'completed')}
                            style={{ padding: '4px 10px', borderRadius: 4, border: 'none', background: '#22c55e', color: '#fff', fontSize: 12, cursor: 'pointer' }}>
                            Complete
                          </button>
                        )}
                        {(wo.status === 'open' || wo.status === 'assigned') && (
                          <button onClick={() => handleStatus(wo, 'cancelled')}
                            style={{ padding: '4px 10px', borderRadius: 4, border: 'none', background: '#475569', color: '#fff', fontSize: 12, cursor: 'pointer' }}>
                            Cancel
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </main>

      {assignModal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }}>
          <div style={{ background: '#1e293b', borderRadius: 8, padding: 24, width: 380, border: '1px solid #334155' }}>
            <h3 style={{ fontSize: 16, margin: '0 0 16px' }}>Assign Work Order</h3>
            <p style={{ fontSize: 13, color: '#94a3b8', margin: '0 0 16px' }}>{assignModal.description || assignModal.type}</p>
            <input
              placeholder="Engineer name"
              value={assignee}
              onChange={e => setAssignee(e.target.value)}
              autoFocus
              style={{ width: '100%', padding: '10px 12px', borderRadius: 6, border: '1px solid #334155', background: '#0f172a', color: '#e2e8f0', fontSize: 14, marginBottom: 12, boxSizing: 'border-box' }}
            />
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
              <button onClick={() => setAssignModal(null)}
                style={{ padding: '8px 16px', borderRadius: 6, border: '1px solid #334155', background: 'transparent', color: '#94a3b8', fontSize: 13, cursor: 'pointer' }}>
                Cancel
              </button>
              <button onClick={handleAssign} disabled={!assignee}
                style={{ padding: '8px 16px', borderRadius: 6, border: 'none', background: assignee ? '#3b82f6' : '#334155', color: assignee ? '#fff' : '#64748b', fontSize: 13, cursor: assignee ? 'pointer' : 'not-allowed' }}>
                Assign
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
