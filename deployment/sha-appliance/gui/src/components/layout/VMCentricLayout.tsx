'use client';

import React, { useState, useEffect } from 'react';
import { usePathname } from 'next/navigation';
import { LeftNavigation } from './LeftNavigation';
import { MainContentArea } from './MainContentArea';
import { RightContextPanel } from './RightContextPanel';
import { NotificationProvider } from '../ui/NotificationSystem';

export interface VMCentricLayoutProps {
  children?: React.ReactNode;
}

export interface NavigationSection {
  id: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  href: string;
}

export const VMCentricLayout = React.memo(({ children }: VMCentricLayoutProps) => {
  const pathname = usePathname();
  const [selectedVM, setSelectedVM] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<string>('virtual-machines');
  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(false);

  // Set active section based on current pathname
  useEffect(() => {
    const path = pathname.replace('/', '') || 'dashboard';
    setActiveSection(path);
  }, [pathname]);

  const handleVMSelect = React.useCallback((vmName: string | null) => {
    setSelectedVM(vmName);
  }, []);

  const handleSectionChange = React.useCallback((section: string) => {
    setActiveSection(section);
  }, []);

  const handleSidebarToggle = React.useCallback(() => {
    setSidebarCollapsed(prev => !prev);
  }, []);

  return (
    <NotificationProvider>
      <div className="flex min-h-screen bg-gradient-to-br from-slate-950 via-gray-900 to-slate-950">
        {/* Left Navigation Panel */}
        <div className={`${sidebarCollapsed ? 'w-16' : 'w-64'} flex-shrink-0 transition-all duration-300`}>
          <LeftNavigation
            activeSection={activeSection}
            onSectionChange={handleSectionChange}
            collapsed={sidebarCollapsed}
            onToggle={handleSidebarToggle}
          />
        </div>

        {/* Main Content Area */}
        <div className="flex-1 flex overflow-hidden">
          <div className="flex-1 overflow-auto p-6">
            <MainContentArea
              section={activeSection}
              selectedVM={selectedVM}
              onVMSelect={handleVMSelect}
            >
              {children}
            </MainContentArea>
          </div>

          {/* Right Context Panel */}
          <div className="w-80 flex-shrink-0 border-l border-gray-700/50 bg-slate-900/50 backdrop-blur-sm">
            <RightContextPanel
              selectedVM={selectedVM}
              onVMSelect={handleVMSelect}
            />
          </div>
        </div>
      </div>
    </NotificationProvider>
  );
});

VMCentricLayout.displayName = 'VMCentricLayout';
