'use client';

import React from 'react';

interface ChartData {
  name: string;
  value: number;
  color: string;
}

interface SimpleChartProps {
  data: ChartData[];
  title?: string;
}

export default function SimpleChart({ data, title }: SimpleChartProps) {
  const total = data.reduce((sum, item) => sum + item.value, 0);

  return (
    <div className="space-y-4">
      {title && <h3 className="text-lg font-semibold">{title}</h3>}
      
      {/* Progress bars chart */}
      <div className="space-y-3">
        {data.map((item, index) => {
          const percentage = total > 0 ? Math.round((item.value / total) * 100) : 0;
          
          return (
            <div key={index} className="space-y-1">
              <div className="flex justify-between text-sm">
                <span className="font-medium">{item.name}</span>
                <span className="text-gray-500">{item.value} ({percentage}%)</span>
              </div>
              <div className="w-full bg-gray-200 rounded-full h-2 dark:bg-gray-700">
                <div
                  className="h-2 rounded-full transition-all duration-300"
                  style={{ 
                    width: `${percentage}%`,
                    backgroundColor: item.color 
                  }}
                />
              </div>
            </div>
          );
        })}
      </div>

      {/* Summary grid */}
      <div className="grid grid-cols-2 gap-4 mt-4">
        {data.map((item, index) => (
          <div key={index} className="text-center p-3 border rounded-lg">
            <div className="text-xl font-bold" style={{ color: item.color }}>
              {item.value}
            </div>
            <div className="text-sm text-gray-500">{item.name}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
