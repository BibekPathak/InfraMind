'use client'

import { useEffect, useState } from 'react'
import ReactEChartsCore from 'echarts-for-react/lib/core'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { TelemetryPoint } from '@/lib/api'

echarts.use([LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

interface Props {
  data: TelemetryPoint[]
}

export default function TelemetryChart({ data }: Props) {
  const [isClient, setIsClient] = useState(false)
  useEffect(() => setIsClient(true), [])

  if (!isClient) return <div style={{ height: 400, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#64748b' }}>Loading chart...</div>

  const times = data.map(d => new Date(d.time).toLocaleTimeString()).reverse()
  const temps = data.map(d => d.temperature).reverse()
  const currents = data.map(d => d.current).reverse()
  const voltages = data.map(d => d.voltage / 100).reverse()

  const option = {
    tooltip: { trigger: 'axis' as const },
    legend: { data: ['Temperature (°C)', 'Current (A)', 'Voltage (×100V)'], textStyle: { color: '#94a3b8' } },
    grid: { left: 60, right: 20, top: 40, bottom: 30 },
    xAxis: { type: 'category' as const, data: times, axisLabel: { color: '#64748b', fontSize: 11 } },
    yAxis: { type: 'value' as const, axisLabel: { color: '#64748b' }, splitLine: { lineStyle: { color: '#1e293b' } } },
    series: [
      { name: 'Temperature (°C)', type: 'line', data: temps, smooth: true, symbol: 'none', lineStyle: { width: 2 }, itemStyle: { color: '#ef4444' } },
      { name: 'Current (A)', type: 'line', data: currents, smooth: true, symbol: 'none', lineStyle: { width: 2 }, itemStyle: { color: '#f59e0b' } },
      { name: 'Voltage (×100V)', type: 'line', data: voltages, smooth: true, symbol: 'none', lineStyle: { width: 2 }, itemStyle: { color: '#38bdf8' } },
    ],
  }

  return <ReactEChartsCore echarts={echarts} option={option} style={{ height: 400 }} notMerge />
}
