'use client'

import { useEffect, useRef, useState, useCallback } from 'react'
import { TelemetryPoint } from '@/lib/api'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
const WS_URL = API_URL.replace(/^http/, 'ws')

export interface WSEvent {
  type: string
  timestamp: string
  asset_id: string
  payload: any
}

interface Props {
  deviceId: string
  onTelemetry?: (point: TelemetryPoint) => void
  onTwinUpdate?: (data: any) => void
  enabled?: boolean
}

export default function LiveStream({ deviceId, onTelemetry, onTwinUpdate, enabled = true }: Props) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout>>()
  const [connected, setConnected] = useState(false)
  const onTelemetryRef = useRef(onTelemetry)
  const onTwinUpdateRef = useRef(onTwinUpdate)
  onTelemetryRef.current = onTelemetry
  onTwinUpdateRef.current = onTwinUpdate

  const connect = useCallback(() => {
    if (!enabled) return

    const url = `${WS_URL}/api/v1/telemetry/ws?device_id=${deviceId}`
    const ws = new WebSocket(url)

    ws.onopen = () => {
      setConnected(true)
    }

    ws.onmessage = (event) => {
      try {
        const evt: WSEvent = JSON.parse(event.data)
        if (evt.type === 'telemetry.updated' && evt.payload && onTelemetryRef.current) {
          onTelemetryRef.current(evt.payload)
        }
        if (evt.type === 'twin.updated' && evt.payload && onTwinUpdateRef.current) {
          onTwinUpdateRef.current(evt.payload)
        }
      } catch {
        // ignore parse errors
      }
    }

    ws.onclose = () => {
      setConnected(false)
      wsRef.current = null
      reconnectTimeoutRef.current = setTimeout(connect, 3000)
    }

    ws.onerror = () => {
      ws.close()
    }

    wsRef.current = ws
  }, [deviceId, enabled])

  useEffect(() => {
    connect()
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [connect])

  return null
}
