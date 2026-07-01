import {
  PieChart, Pie, Cell, Legend, Tooltip, ResponsiveContainer,
} from 'recharts'

// Status labels, display names, and colours in one place
const SLICES = [
  { key: 'pending',   label: 'Pending',   color: '#f59e0b' },
  { key: 'running',   label: 'Running',   color: '#3b82f6' },
  { key: 'completed', label: 'Completed', color: '#10b981' },
  { key: 'failed',    label: 'Failed',    color: '#ef4444' },
] as const

interface StatusPieChartProps {
  pending: number
  running: number
  completed: number
  failed: number
}

// Pie chart showing the distribution of pipeline jobs across status categories
export function StatusPieChart({ pending, running, completed, failed }: StatusPieChartProps) {
  const counts = { pending, running, completed, failed }
  const data = SLICES
    .map(s => ({ name: s.label, value: counts[s.key], color: s.color }))
    .filter(d => d.value > 0)

  if (data.length === 0) {
    return <p className="text-center text-gray-400 text-sm py-8">No jobs yet</p>
  }

  return (
    <ResponsiveContainer width="100%" height={240}>
      <PieChart>
        <Pie data={data} cx="50%" cy="45%" outerRadius={80} dataKey="value" label={({ name, percent }) =>
          `${name} ${(percent * 100).toFixed(0)}%`
        } labelLine={false}>
          {data.map((entry, i) => (
            <Cell key={i} fill={entry.color} />
          ))}
        </Pie>
        <Tooltip formatter={(v: number) => [v, 'Jobs']} contentStyle={{ fontSize: 12, borderRadius: 8 }} />
        <Legend wrapperStyle={{ fontSize: 12 }} />
      </PieChart>
    </ResponsiveContainer>
  )
}
