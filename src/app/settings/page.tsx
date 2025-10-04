'use client';

import { useState } from 'react';
import { HiCog, HiKey, HiServer } from 'react-icons/hi';
import OSSEASettings from './ossea/page';
import { VMAEnrollmentManager } from '@/components/settings/VMAEnrollmentManager';

export default function Settings() {
  const [activeTab, setActiveTab] = useState('ossea');

  const tabs = [
    { id: 'ossea', name: 'OSSEA Configuration', icon: HiCog },
    { id: 'vmware', name: 'VMware Credentials', icon: HiKey },
    { id: 'vma', name: 'VMA Enrollment', icon: HiServer },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
          Settings
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Configure OSSEA connection, VMware credentials, and VMA enrollment system
        </p>
      </div>

      {/* Custom Tab Navigation */}
      <div className="border-b border-gray-200 dark:border-gray-700 mb-6">
        <nav className="-mb-px flex space-x-8">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`group inline-flex items-center py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                }`}
              >
                <Icon className="h-5 w-5 mr-2" />
                {tab.name}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="mt-4">
        {activeTab === 'ossea' && <OSSEASettings />}
        {activeTab === 'vmware' && (
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-6">
            <div className="flex items-center">
              <HiKey className="h-5 w-5 text-blue-500 mr-3" />
              <div>
                <h3 className="text-lg font-medium text-blue-900 dark:text-blue-100">
                  VMware Credentials Management
                </h3>
                <p className="text-blue-700 dark:text-blue-300 text-sm mt-1">
                  VMware credentials are managed through the existing interface. This section will be enhanced in a future update.
                </p>
              </div>
            </div>
          </div>
        )}
        {activeTab === 'vma' && <VMAEnrollmentManager />}
      </div>
    </div>
  );
}