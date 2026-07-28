interface Props {
  score: number
  level: string
}

export default function HealthBadge({ score, level }: Props) {
  const color = level === 'healthy' ? '#22c55e' : level === 'warning' ? '#f59e0b' : '#ef4444'

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, background: '#1e293b', borderRadius: 8, padding: '12px 16px', border: '1px solid #334155' }}>
      <div style={{ position: 'relative', width: 56, height: 56 }}>
        <svg width="56" height="56" viewBox="0 0 56 56">
          <circle cx="28" cy="28" r="24" fill="none" stroke="#334155" strokeWidth="4" />
          <circle cx="28" cy="28" r="24" fill="none" stroke={color} strokeWidth="4"
            strokeDasharray={`${(score / 100) * 150.8} 150.8`}
            transform="rotate(-90 28 28)"
            strokeLinecap="round" />
        </svg>
        <div style={{ position: 'absolute', top: 0, left: 0, width: 56, height: 56, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14, fontWeight: 700 }}>
          {Math.round(score)}
        </div>
      </div>
      <div>
        <div style={{ fontSize: 14, fontWeight: 600, color }}>{level.toUpperCase()}</div>
        <div style={{ fontSize: 11, color: '#64748b' }}>Health Score</div>
      </div>
    </div>
  )
}
