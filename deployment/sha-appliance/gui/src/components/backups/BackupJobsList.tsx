'use client';

import { Card, Spinner, Table, Badge, Button } from 'flowbite-react';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { BackupJob } from '@/lib/types';
import { HiRefresh, HiExclamationCircle, HiArchive } from 'react-icons/hi';
import { BackupJobRow } from './BackupJobRow';

interface BackupJobsListProps {
  vmName?: string;
  repositoryId?: string;
  status?: string;
  onBackupSelect?: (backup: BackupJob) => void;
  onBrowseFiles?: (backup: BackupJob) => void;
}

export function BackupJobsList({
  vmName,
  repositoryId,
  status,
  onBackupSelect,
  onBrowseFiles
}: BackupJobsListProps) {
  // Fetch backups using React Query
  const {
    data: backupsData,
    isLoading,
    error,
    refetch,
    isRefetching
  } = useQuery({
    queryKey: ['backups', vmName, repositoryId, status],
    queryFn: () => api.listBackups({
      vm_name: vmName,
      repository_id: repositoryId,
      status: status
    }),
    refetchInterval: 5000, // Refresh every 5 seconds for real-time updates
  });

  const backups = backupsData?.backups || [];

  if (isLoading) {
    return (
      <Card>
        <div className="flex items-center justify-center p-8">
          <Spinner size="lg" />
          <span className="ml-3 text-gray-600 dark:text-gray-400">
            Loading backups...
          </span>
        </div>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <div className="text-center p-8">
          <HiExclamationCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
            Error Loading Backups
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {error instanceof Error ? error.message : 'Failed to load backups'}
          </p>
          <Button onClick={() => refetch()} color="blue">
            <HiRefresh className="w-4 h-4 mr-2" />
            Retry
          </Button>
        </div>
      </Card>
    );
  }

  if (backups.length === 0) {
    return (
      <Card>
        <div className="text-center p-12">
          <HiArchive className="w-16 h-16 text-gray-400 dark:text-gray-600 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
            No Backups Found
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            {vmName ? `No backups found for ${vmName}` : 'No backups available. Start a backup to get started.'}
          </p>
        </div>
      </Card>
    );
  }

  return (
    <Card>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
            Backup Jobs
          </h3>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {backups.length} {backups.length === 1 ? 'backup' : 'backups'} found
          </p>
        </div>
        <Button
          size="sm"
          color="gray"
          onClick={() => refetch()}
          disabled={isRefetching}
        >
          <HiRefresh className={`w-4 h-4 mr-2 ${isRefetching ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
      </div>

      <div className="overflow-x-auto">
        <Table>
          <Table.Head>
            <Table.HeadCell>VM Name</Table.HeadCell>
            <Table.HeadCell>Disk</Table.HeadCell>
            <Table.HeadCell>Type</Table.HeadCell>
            <Table.HeadCell>Repository</Table.HeadCell>
            <Table.HeadCell>Status</Table.HeadCell>
            <Table.HeadCell>Progress</Table.HeadCell>
            <Table.HeadCell>Size</Table.HeadCell>
            <Table.HeadCell>Created</Table.HeadCell>
            <Table.HeadCell>Actions</Table.HeadCell>
          </Table.Head>
          <Table.Body className="divide-y">
            {backups.map((backup) => (
              <BackupJobRow
                key={backup.backup_id}
                backup={backup}
                onSelect={onBackupSelect}
                onBrowseFiles={onBrowseFiles}
              />
            ))}
          </Table.Body>
        </Table>
      </div>
    </Card>
  );
}
