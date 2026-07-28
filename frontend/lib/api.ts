const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export interface Asset {
  id: string
  name: string
  type: string
  location?: Record<string, unknown>
  metadata?: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface Device {
  id: string
  assetId: string
  firmwareVersion: string
  status: string
  location?: Record<string, unknown>
  lastHeartbeat?: string
  createdAt: string
  updatedAt: string
}

export interface TelemetryPoint {
  time: string
  deviceId: string
  temperature: number
  current: number
  voltage: number
  humidity: number
}

export interface HealthScore {
  score: number
  level: 'healthy' | 'warning' | 'critical'
  factors: { name: string; impact: number }[]
}

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`)
  return res.json()
}

export async function getAssets(): Promise<Asset[]> {
  return fetchJSON<Asset[]>(`${API_URL}/api/v1/assets`)
}

export async function getAsset(id: string): Promise<Asset> {
  return fetchJSON<Asset>(`${API_URL}/api/v1/assets/${id}`)
}

export async function getDevices(assetId: string): Promise<Device[]> {
  return fetchJSON<Device[]>(`${API_URL}/api/v1/assets/${assetId}/devices`)
}

export async function getDevice(id: string): Promise<Device> {
  return fetchJSON<Device>(`${API_URL}/api/v1/device/${id}`)
}

export async function getTelemetry(deviceId: string, from?: string, to?: string): Promise<TelemetryPoint[]> {
  const params = new URLSearchParams()
  if (from) params.set('from', from)
  if (to) params.set('to', to)
  const qs = params.toString()
  return fetchJSON<TelemetryPoint[]>(`${API_URL}/api/v1/devices/${deviceId}/telemetry${qs ? '?' + qs : ''}`)
}

export async function getLiveTelemetry(deviceId: string): Promise<TelemetryPoint> {
  return fetchJSON<TelemetryPoint>(`${API_URL}/api/v1/telemetry/live?device_id=${deviceId}`)
}

export async function getHealth(deviceId: string, telemetry: TelemetryPoint): Promise<HealthScore> {
  const params = new URLSearchParams({
    temperature: String(telemetry.temperature),
    current: String(telemetry.current),
    voltage: String(telemetry.voltage),
    humidity: String(telemetry.humidity),
  })
  return fetchJSON<HealthScore>(`${API_URL}/api/v1/health/${deviceId}?${params}`)
}
