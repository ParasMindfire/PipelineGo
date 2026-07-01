import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from 'recharts'
import type { RateDataPoint } from '../../types/metrics'

interface RateChartProps { data: RateDataPoint[] }

// Line chart plotting records-per-second over the last 30 polling intervals
export function RateChart({ data }: RateChartProps) {
  if (data.length < 2) {
    return <p className="text-center text-gray-400 text-sm py-8">Collecting data…</p>
  }
  return (
    <ResponsiveContainer width="100%" height={220}>
      <LineChart data={data} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
        <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#94a3b8' }} />
        <YAxis tick={{ fontSize: 10, fill: '#94a3b8' }} unit=" r/s" width={55} />
        <Tooltip
          formatter={(v: number) => [`${v} rec/s`, 'Processing rate']}
          contentStyle={{ fontSize: 12, borderRadius: 8 }}
        />
        <Line
          type="monotone" dataKey="rate" stroke="#3b82f6"
          strokeWidth={2} dot={false} activeDot={{ r: 4 }}
        />
      </LineChart>
    </ResponsiveContainer>
  )
}
