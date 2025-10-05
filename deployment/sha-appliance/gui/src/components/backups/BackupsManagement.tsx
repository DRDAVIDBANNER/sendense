'use client';

import { HiArchive, HiCheckCircle, HiClock } from 'react-icons/hi';
import { Card } from 'flowbite-react';

export function BackupsManagement() {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white flex items-center">
            <HiArchive className="w-8 h-8 mr-3 text-blue-600" />
            Backup Management
          </h1>
          <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
            Manage VM backups, monitor progress, and restore individual files
          </p>
        </div>
      </div>

      {/* Statistics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card>
          <div className="flex items-center">
            <div className="p-3 rounded-full bg-blue-100 dark:bg-blue-900">
              <HiArchive className="w-6 h-6 text-blue-600 dark:text-blue-400" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600 dark:text-gray-400">Total Backups</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">-</p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="p-3 rounded-full bg-green-100 dark:bg-green-900">
              <HiCheckCircle className="w-6 h-6 text-green-600 dark:text-green-400" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600 dark:text-gray-400">Completed</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">-</p>
            </div>
          </div>
        </Card>

        <Card>
          <div className="flex items-center">
            <div className="p-3 rounded-full bg-yellow-100 dark:bg-yellow-900">
              <HiClock className="w-6 h-6 text-yellow-600 dark:text-yellow-400" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600 dark:text-gray-400">In Progress</p>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">-</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Main Content Card */}
      <Card>
        <div className="text-center py-12">
          <HiArchive className="w-16 h-16 mx-auto text-gray-400 dark:text-gray-600 mb-4" />
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
            Backup Management Ready
          </h3>
          <p className="text-gray-600 dark:text-gray-400 mb-6 max-w-md mx-auto">
            Backup infrastructure is operational. The backup jobs list and controls will be implemented in the next phase.
          </p>
          <div className="space-y-3 max-w-md mx-auto text-left">
            <div className="flex items-start">
              <HiCheckCircle className="w-5 h-5 text-green-600 mt-0.5 mr-2" />
              <div>
                <p className="text-sm font-medium text-gray-900 dark:text-white">Task 5: Backup API Endpoints</p>
                <p className="text-xs text-gray-600 dark:text-gray-400">Complete - 5 REST endpoints operational</p>
              </div>
            </div>
            <div className="flex items-start">
              <HiCheckCircle className="w-5 h-5 text-green-600 mt-0.5 mr-2" />
              <div>
                <p className="text-sm font-medium text-gray-900 dark:text-white">Task 4: File-Level Restore</p>
                <p className="text-xs text-gray-600 dark:text-gray-400">Complete - 9 REST endpoints operational</p>
              </div>
            </div>
            <div className="flex items-start">
              <HiClock className="w-5 h-5 text-yellow-600 mt-0.5 mr-2" />
              <div>
                <p className="text-sm font-medium text-gray-900 dark:text-white">GUI Integration</p>
                <p className="text-xs text-gray-600 dark:text-gray-400">Phase 1 - Navigation complete, components next</p>
              </div>
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
}
