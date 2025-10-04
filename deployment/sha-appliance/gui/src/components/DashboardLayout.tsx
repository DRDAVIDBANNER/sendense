'use client';

import { AppSidebar } from './Sidebar';

interface DashboardLayoutProps {
  children: React.ReactNode;
  currentPage: string;
}

export function DashboardLayout({ children, currentPage }: DashboardLayoutProps) {
  return (
    <div className="flex min-h-screen">
      <div className="w-64 flex-shrink-0">
        <AppSidebar currentPage={currentPage} />
      </div>
      <main className="flex-1 overflow-auto">
        {children}
      </main>
    </div>
  );
}