'use client';

import React from 'react';
import { LeftNavigation } from '@/components/layout/LeftNavigation';
import HistoricalAnalytics from '@/components/analytics/HistoricalAnalytics';

export default function AnalyticsPage() {
  return (
    <div className="flex min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Left Navigation */}
      <div className="w-64 flex-shrink-0">
        <LeftNavigation currentPage="analytics" />
      </div>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        <div className="p-6">
          <HistoricalAnalytics />
        </div>
      </main>
    </div>
  );
}
