'use client';

import { useState } from 'react';
import { HiArchive, HiCheckCircle, HiClock, HiPlus } from 'react-icons/hi';
import { Card, Button } from 'flowbite-react';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { BackupJob } from '@/lib/types';
import { BackupJobsList } from './BackupJobsList';
import { StartBackupModal } from './StartBackupModal';
import { FileBrowserModal } from './FileBrowserModal';

export function BackupsManagement() {
  const [selectedBackup, setSelectedBackup] = useState<BackupJob | null>(null);
  const [showStartBackupModal, setShowStartBackupModal] = useState(false);
  const [browseFilesBackup, setBrowseFilesBackup] = useState<BackupJob | null>(null);

  // Fetch all backups for statistics
  const { data: backupsData } = useQuery({
    queryKey: ['backups-all'],
    queryFn: () => api.listBackups({}),
    refetchInterval: 10000, // Refresh every 10 seconds
  });

  const allBackups = backupsData?.backups || [];
  const completedBackups = allBackups.filter(b => b.status === 'completed').length;
  const runningBackups = allBackups.filter(b => b.status === 'running').length;

  const handleBrowseFiles = (backup: BackupJob) => {
    setBrowseFilesBackup(backup);
  };

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
        <Button color="blue" onClick={() => setShowStartBackupModal(true)}>
          <HiPlus className="w-4 h-4 mr-2" />
          Start Backup
        </Button>
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
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {allBackups.length}
              </p>
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
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {completedBackups}
              </p>
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
              <p className="text-2xl font-bold text-gray-900 dark:text-white">
                {runningBackups}
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* Backup Jobs List */}
      <BackupJobsList
        onBackupSelect={setSelectedBackup}
        onBrowseFiles={handleBrowseFiles}
      />

      {/* Start Backup Modal */}
      <StartBackupModal
        isOpen={showStartBackupModal}
        onClose={() => setShowStartBackupModal(false)}
      />

      {/* File Browser Modal */}
      <FileBrowserModal
        isOpen={!!browseFilesBackup}
        onClose={() => setBrowseFilesBackup(null)}
        backup={browseFilesBackup}
      />
    </div>
  );
}
