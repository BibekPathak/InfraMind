import Link from 'next/link'

export default function Nav() {
  return (
    <nav style={{ padding: '16px 24px', borderBottom: '1px solid #334155', background: '#1e293b' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 32, maxWidth: 1200, margin: '0 auto' }}>
        <Link href="/" style={{ fontWeight: 700, fontSize: 20, color: '#38bdf8', textDecoration: 'none' }}>
          InfraMind
        </Link>
        <Link href="/" style={{ color: '#94a3b8', textDecoration: 'none', fontSize: 14 }}>Dashboard</Link>
        <Link href="/devices" style={{ color: '#94a3b8', textDecoration: 'none', fontSize: 14 }}>Devices</Link>
        <Link href="/telemetry" style={{ color: '#94a3b8', textDecoration: 'none', fontSize: 14 }}>Telemetry</Link>
        <Link href="/alerts" style={{ color: '#94a3b8', textDecoration: 'none', fontSize: 14 }}>Alerts</Link>
        <Link href="/twins" style={{ color: '#94a3b8', textDecoration: 'none', fontSize: 14 }}>Twins</Link>
        <Link href="/ai" style={{ color: '#94a3b8', textDecoration: 'none', fontSize: 14 }}>AI</Link>
        <Link href="/operations" style={{ color: '#94a3b8', textDecoration: 'none', fontSize: 14 }}>Operations</Link>
      </div>
    </nav>
  )
}
