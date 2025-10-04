'use client';

import React from 'react';
import { LeftNavigation } from '@/components/layout/LeftNavigation';
import { DiscoveryView } from '@/components/discovery/DiscoveryView';

export default function DiscoveryPage() {
  // Handle VM selection - navigate to virtual machines page with selection
  const handleVMSelect = (vmName: string) => {
    window.location.href = `/virtual-machines?selected=${encodeURIComponent(vmName)}`;
  };

  return (
    <div className="flex min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Left Navigation */}
      <div className="w-64 flex-shrink-0">
        <LeftNavigation currentPage="discovery" />
      </div>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        <DiscoveryView onVMSelect={handleVMSelect} />
      </main>
    </div>
  );
}
