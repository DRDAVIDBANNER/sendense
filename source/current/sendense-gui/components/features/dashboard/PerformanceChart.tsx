"use client";

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Area, AreaChart } from 'recharts';

const performanceData = [
  { time: '00:00', throughput: 120, jobs: 2, errors: 0 },
  { time: '04:00', throughput: 95, jobs: 1, errors: 0 },
  { time: '08:00', throughput: 180, jobs: 4, errors: 1 },
  { time: '12:00', throughput: 220, jobs: 5, errors: 0 },
  { time: '16:00', throughput: 195, jobs: 4, errors: 0 },
  { time: '20:00', throughput: 150, jobs: 3, errors: 0 },
];

interface PerformanceChartProps {
  className?: string;
}

export function PerformanceChart({ className }: PerformanceChartProps) {
  return (
    <div className={className}>
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={performanceData}>
            <defs>
              <linearGradient id="throughputGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#023E8A" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#023E8A" stopOpacity={0.1}/>
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
            <XAxis
              dataKey="time"
              stroke="#9ca3af"
              fontSize={12}
              tickLine={false}
              axisLine={false}
            />
            <YAxis
              stroke="#9ca3af"
              fontSize={12}
              tickLine={false}
              axisLine={false}
              label={{ value: 'MB/s', angle: -90, position: 'insideLeft' }}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: '#12172a',
                border: '1px solid #374151',
                borderRadius: '8px',
                color: '#e4e7eb'
              }}
              labelStyle={{ color: '#e4e7eb' }}
            />
            <Area
              type="monotone"
              dataKey="throughput"
              stroke="#023E8A"
              strokeWidth={2}
              fill="url(#throughputGradient)"
              name="Throughput (MB/s)"
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      <div className="flex items-center justify-center gap-6 mt-4 text-sm">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-primary"></div>
          <span className="text-muted-foreground">Throughput</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground">Peak: 220 MB/s</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground">Avg: 160 MB/s</span>
        </div>
      </div>
    </div>
  );
}
