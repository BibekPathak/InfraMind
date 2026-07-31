'use client'

import { useEffect, useState, useCallback } from 'react'
import Nav from '@/components/Nav'
import HealthBadge from '@/components/HealthBadge'
import LiveStream from '@/components/LiveStream'
import { getLiveTelemetry, TelemetryPoint } from '@/lib/api'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
const DEVICE_ID = 'tx-001'

interface HealthFactor {
  name: string
  impact: number
  details: string
}

interface Anomaly {
  metric: string
  severity: string
  description: string
  value: number
  expected: number
}

interface FailurePrediction {
  timeToWarningHours: number | null
  timeToCriticalHours: number | null
  confidence: number
  trendDirection: string
}

interface Recommendation {
  priority: string
  action: string
  reason: string
  estimatedCost: string
}

interface Analysis {
  healthScore: number
  healthLevel: string
  healthFactors: HealthFactor[]
  anomalies: Anomaly[]
  failurePrediction: FailurePrediction | null
  recommendations: Recommendation[]
}

export default function AIPage() {
  const [analysis, setAnalysis] = useState<Analysis | null>(null)
  const [live, setLive] = useState<TelemetryPoint | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const fetchAnalysis = useCallback(async (t: TelemetryPoint) => {
    try {
      const params = new URLSearchParams({
        temperature: String(t.temperature),
        current: String(t.current),
        voltage: String(t.voltage),
        humidity: String(t.humidity),
      })
      const res = await fetch(`${API_URL}/api/v1/health/${DEVICE_ID}/analysis?${params}`)
      if (res.ok) {
        setAnalysis(await res.json())
        setError('')
      }
    } catch {
      setError('AI service unavailable')
    }
  }, [])

  useEffect(() => {
    const load = async () => {
      try {
        const t = await getLiveTelemetry(DEVICE_ID)
        setLive(t)
        await fetchAnalysis(t)
      } catch {
        setError('No live telemetry available')
      }
      setLoading(false)
    }
    load()
  }, [fetchAnalysis])

  const handleTelemetry = useCallback((t: TelemetryPoint) => {
    setLive(t)
    fetchAnalysis(t)
  }, [fetchAnalysis])

  const severityColor = (s: string) => s === 'critical' ? '#ef4444' : s === 'warning' ? '#f59e0b' : '#3b82f6'
  const priorityColor = (p: string) => p === 'critical' ? '#ef4444' : p === 'high' ? '#f59e0b' : p === 'medium' ? '#3b82f6' : '#64748b'

  return (
    <div style={{ minHeight: '100vh', background: '#0f172a' }}>
      <LiveStream deviceId={DEVICE_ID} onTelemetry={handleTelemetry} />
      <Nav />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: 24 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <h1 style={{ fontSize: 24, margin: 0 }}>AI Analysis</h1>
          {analysis && <HealthBadge score={analysis.healthScore} level={analysis.healthLevel} />}
        </div>

        {error && <p style={{ color: '#f59e0b', marginBottom: 16 }}>{error}</p>}

        {loading ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>Loading AI analysis...</div>
        ) : !analysis ? (
          <div style={{ textAlign: 'center', padding: 48, color: '#64748b' }}>No analysis available.</div>
        ) : (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 24 }}>
              <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
                <h3 style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 12px' }}>HEALTH FACTORS</h3>
                {analysis.healthFactors.length === 0 ? (
                  <p style={{ color: '#64748b', fontSize: 13 }}>No contributing factors.</p>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                    {analysis.healthFactors.map(f => (
                      <div key={f.name} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 13 }}>
                        <div>
                          <span style={{ color: '#e2e8f0' }}>{f.name.replace('_', ' ')}</span>
                          {f.details && <div style={{ color: '#64748b', fontSize: 11 }}>{f.details}</div>}
                        </div>
                        <span style={{ color: f.impact < 0 ? '#ef4444' : '#22c55e', fontWeight: 600 }}>{f.impact > 0 ? '+' : ''}{f.impact}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
                <h3 style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 12px' }}>ANOMALIES</h3>
                {analysis.anomalies.length === 0 ? (
                  <p style={{ color: '#22c55e', fontSize: 13 }}>No anomalies detected.</p>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 8, maxHeight: 200, overflowY: 'auto' }}>
                    {analysis.anomalies.map((a, i) => (
                      <div key={i} style={{ fontSize: 13, padding: '8px', borderRadius: 6, background: '#0f172a', border: '1px solid #334155' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                          <span style={{ color: severityColor(a.severity), fontWeight: 600, textTransform: 'uppercase', fontSize: 11 }}>{a.severity}</span>
                          <span style={{ color: '#64748b', fontSize: 11 }}>{a.metric}</span>
                        </div>
                        <div style={{ color: '#e2e8f0', marginTop: 4 }}>{a.description}</div>
                        <div style={{ color: '#64748b', fontSize: 11, marginTop: 2 }}>value: {a.value} / expected: {a.expected}</div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {analysis.failurePrediction && (
              <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155', marginBottom: 24 }}>
                <h3 style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 12px' }}>FAILURE PREDICTION</h3>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 12 }}>
                  <div>
                    <div style={{ fontSize: 11, color: '#64748b' }}>Trend Direction</div>
                    <div style={{ fontSize: 16, fontWeight: 700, color: analysis.failurePrediction.trendDirection === 'rising' ? '#f59e0b' : '#22c55e', textTransform: 'capitalize' }}>
                      {analysis.failurePrediction.trendDirection}
                    </div>
                  </div>
                  <div>
                    <div style={{ fontSize: 11, color: '#64748b' }}>Time to Warning</div>
                    <div style={{ fontSize: 16, fontWeight: 700, color: '#e2e8f0' }}>
                      {analysis.failurePrediction.timeToWarningHours != null ? `${analysis.failurePrediction.timeToWarningHours.toFixed(1)}h` : '--'}
                    </div>
                  </div>
                  <div>
                    <div style={{ fontSize: 11, color: '#64748b' }}>Time to Critical</div>
                    <div style={{ fontSize: 16, fontWeight: 700, color: '#ef4444' }}>
                      {analysis.failurePrediction.timeToCriticalHours != null ? `${analysis.failurePrediction.timeToCriticalHours.toFixed(1)}h` : '--'}
                    </div>
                  </div>
                  <div>
                    <div style={{ fontSize: 11, color: '#64748b' }}>Confidence</div>
                    <div style={{ fontSize: 16, fontWeight: 700, color: '#e2e8f0' }}>
                      {Math.round(analysis.failurePrediction.confidence * 100)}%
                    </div>
                  </div>
                </div>
              </div>
            )}

            <div style={{ background: '#1e293b', borderRadius: 8, padding: 16, border: '1px solid #334155' }}>
              <h3 style={{ fontSize: 14, color: '#94a3b8', margin: '0 0 12px' }}>RECOMMENDATIONS</h3>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                {analysis.recommendations.map((rec, i) => (
                  <div key={i} style={{ display: 'flex', gap: 12, alignItems: 'flex-start', padding: '10px', borderRadius: 6, background: '#0f172a', border: '1px solid #334155' }}>
                    <span style={{
                      padding: '2px 8px', borderRadius: 4, fontSize: 11, fontWeight: 600, textTransform: 'uppercase', whiteSpace: 'nowrap',
                      background: priorityColor(rec.priority) + '22', color: priorityColor(rec.priority)
                    }}>
                      {rec.priority}
                    </span>
                    <div style={{ flex: 1 }}>
                      <div style={{ fontSize: 14, fontWeight: 600, color: '#e2e8f0' }}>{rec.action}</div>
                      <div style={{ fontSize: 12, color: '#94a3b8', marginTop: 2 }}>{rec.reason}</div>
                    </div>
                    <span style={{ fontSize: 12, color: '#64748b', whiteSpace: 'nowrap' }}>{rec.estimatedCost}</span>
                  </div>
                ))}
              </div>
            </div>
          </>
        )}
      </main>
    </div>
  )
}
