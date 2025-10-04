// Three Panel Layout Component - Reavyr-inspired design
// Following our best practices: TypeScript, responsive, accessible

'use client';

import React, { useState, useCallback } from 'react';
import { LeftNavigation } from './LeftNavigation';
import { RightContextPanel } from './RightContextPanel';
import { VMContextDetails } from '@/lib/types';

interface ThreePanelLayoutProps {
  children: React.ReactNode;
  selectedVM?: string | null;
  vmContext?: VMContextDetails;
  vmContextLoading?: boolean;
  vmContextError?: string;
  onNetworkMapping?: () => void;
  onStartReplication?: () => void;
  onLiveFailover?: () => void;
  onTestFailover?: () => void;
  onCleanup?: () => void;
  rightPanelCollapsed?: boolean;
  onToggleRightPanel?: () => void;
}

export const ThreePanelLayout = React.memo(({
  children,
  selectedVM,
  vmContext,
  vmContextLoading = false,
  vmContextError,
  onNetworkMapping,
  onStartReplication,
  onLiveFailover,
  onTestFailover,
  onCleanup,
  rightPanelCollapsed = false,
  onToggleRightPanel
}: ThreePanelLayoutProps) => {
  
  const [leftPanelCollapsed, setLeftPanelCollapsed] = useState(false);

  const toggleLeftPanel = useCallback(() => {
    setLeftPanelCollapsed(prev => !prev);
  }, []);

  const handleToggleRightPanel = useCallback(() => {
    onToggleRightPanel?.();
  }, [onToggleRightPanel]);

  return (
    <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
      {/* Left Navigation Panel */}
      <div className={`flex-shrink-0 transition-all duration-300 ${
        leftPanelCollapsed ? 'w-16' : 'w-64'
      }`}>
        <LeftNavigation />
        
        {/* Left Panel Toggle Button */}
        <button
          onClick={toggleLeftPanel}
          className="absolute top-4 left-4 z-10 p-2 rounded-lg bg-white dark:bg-gray-800 shadow-sm border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700 lg:hidden"
          aria-label={leftPanelCollapsed ? 'Expand navigation' : 'Collapse navigation'}
        >
          <svg
            className={`w-4 h-4 transition-transform ${leftPanelCollapsed ? 'rotate-180' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>
      </div>

      {/* Main Content Area */}
      <div className={`flex-1 flex flex-col min-w-0 transition-all duration-300 ${
        rightPanelCollapsed ? 'mr-0' : 'mr-80'
      }`}>
        {/* Main Content */}
        <main className="flex-1 overflow-auto">
          {children}
        </main>
      </div>

      {/* Right Context Panel */}
      <div className={`fixed right-0 top-0 h-full transition-all duration-300 z-20 ${
        rightPanelCollapsed ? 'translate-x-full' : 'translate-x-0'
      }`}>
        <RightContextPanel
          vmContext={vmContext}
          loading={vmContextLoading}
          error={vmContextError}
          onNetworkMapping={onNetworkMapping}
          onStartReplication={onStartReplication}
          onLiveFailover={onLiveFailover}
          onTestFailover={onTestFailover}
          onCleanup={onCleanup}
          className="h-full shadow-lg"
        />
        
        {/* Right Panel Toggle Button */}
        <button
          onClick={handleToggleRightPanel}
          className={`absolute top-4 -left-10 p-2 rounded-l-lg bg-white dark:bg-gray-800 shadow-sm border border-gray-200 dark:border-gray-700 border-r-0 hover:bg-gray-50 dark:hover:bg-gray-700 transition-all duration-300 ${
            rightPanelCollapsed ? 'translate-x-0' : '-translate-x-0'
          }`}
          aria-label={rightPanelCollapsed ? 'Show context panel' : 'Hide context panel'}
        >
          <svg
            className={`w-4 h-4 transition-transform ${rightPanelCollapsed ? 'rotate-180' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </button>
      </div>

      {/* Mobile Overlay */}
      {!rightPanelCollapsed && (
        <div 
          className="fixed inset-0 bg-black bg-opacity-50 z-10 lg:hidden"
          onClick={handleToggleRightPanel}
          aria-hidden="true"
        />
      )}
    </div>
  );
});

ThreePanelLayout.displayName = 'ThreePanelLayout';
