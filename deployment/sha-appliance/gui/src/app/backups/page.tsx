'use client';

import { AppSidebar } from '@/components/Sidebar';
import { ClientOnly } from '@/components/ClientOnly';
import { BackupsManagement } from '@/components/backups/BackupsManagement';

export default function BackupsPage() {
  return (
    <div className="flex h-screen bg-gray-100 dark:bg-gray-900">
      <ClientOnly>
        <AppSidebar currentPage="backups" />
      </ClientOnly>
      <div className="flex-1 flex flex-col overflow-hidden">
        <main className="flex-1 overflow-x-hidden overflow-y-auto bg-gray-100 dark:bg-gray-900">
          <div className="container mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <BackupsManagement />
          </div>
        </main>
      </div>
    </div>
  );
}
