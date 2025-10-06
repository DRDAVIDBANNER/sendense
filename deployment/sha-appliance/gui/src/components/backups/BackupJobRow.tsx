'use client';

import { Table, Badge, Button, Progress } from 'flowbite-react';
import { BackupJob } from '@/lib/types';
import { HiEye, HiFolder, HiTrash } from 'react-icons/hi';
import { formatDistanceToNow } from 'date-fns';

interface BackupJobRowProps {
  backup: BackupJob;
  onSelect?: (backup: BackupJob) => void;
  onBrowseFiles?: (backup: BackupJob) => void;
  onDelete?: (backup: BackupJob) => void;
}

export function BackupJobRow({ backup, onSelect, onBrowseFiles, onDelete }: BackupJobRowProps) {
  // Format bytes to human-readable format
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  // Calculate progress percentage
  const progressPercent = backup.total_bytes > 0
    ? Math.round((backup.bytes_transferred / backup.total_bytes) * 100)
    : 0;

  // Status badge color mapping
  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'completed':
        return 'success';
      case 'running':
        return 'info';
      case 'pending':
        return 'warning';
      case 'failed':
        return 'failure';
      default:
        return 'gray';
    }
  };

  // Type badge color
  const getTypeColor = (type: string): string => {
    return type === 'full' ? 'blue' : 'purple';
  };

  // Format date to relative time
  const formatDate = (dateString: string): string => {
    try {
      return formatDistanceToNow(new Date(dateString), { addSuffix: true });
    } catch {
      return dateString;
    }
  };

  return (
    <Table.Row className="bg-white dark:border-gray-700 dark:bg-gray-800">
      {/* VM Name */}
      <Table.Cell className="whitespace-nowrap font-medium text-gray-900 dark:text-white">
        {backup.vm_name}
      </Table.Cell>

      {/* Disk ID */}
      <Table.Cell>
        <span className="text-sm text-gray-600 dark:text-gray-400">
          Disk {backup.disk_id}
        </span>
      </Table.Cell>

      {/* Backup Type */}
      <Table.Cell>
        <Badge color={getTypeColor(backup.backup_type)} size="sm">
          {backup.backup_type}
        </Badge>
      </Table.Cell>

      {/* Repository */}
      <Table.Cell>
        <span className="text-sm text-gray-600 dark:text-gray-400">
          {backup.repository_id}
        </span>
      </Table.Cell>

      {/* Status */}
      <Table.Cell>
        <Badge color={getStatusColor(backup.status)} size="sm">
          {backup.status}
        </Badge>
      </Table.Cell>

      {/* Progress */}
      <Table.Cell>
        {backup.status === 'running' ? (
          <div className="w-24">
            <Progress
              progress={progressPercent}
              size="sm"
              color="blue"
              labelProgress
            />
          </div>
        ) : backup.status === 'completed' ? (
          <span className="text-sm text-green-600 dark:text-green-400">100%</span>
        ) : (
          <span className="text-sm text-gray-500 dark:text-gray-400">-</span>
        )}
      </Table.Cell>

      {/* Size */}
      <Table.Cell>
        <div className="text-sm">
          <div className="text-gray-900 dark:text-white">
            {formatBytes(backup.bytes_transferred)}
          </div>
          {backup.total_bytes > 0 && backup.status !== 'completed' && (
            <div className="text-xs text-gray-500 dark:text-gray-400">
              of {formatBytes(backup.total_bytes)}
            </div>
          )}
        </div>
      </Table.Cell>

      {/* Created Date */}
      <Table.Cell>
        <span className="text-sm text-gray-600 dark:text-gray-400">
          {formatDate(backup.created_at)}
        </span>
      </Table.Cell>

      {/* Actions */}
      <Table.Cell>
        <div className="flex items-center gap-2">
          {/* View Details */}
          <Button
            size="xs"
            color="gray"
            onClick={() => onSelect?.(backup)}
            title="View Details"
          >
            <HiEye className="w-4 h-4" />
          </Button>

          {/* Browse Files (only for completed backups) */}
          {backup.status === 'completed' && (
            <Button
              size="xs"
              color="blue"
              onClick={() => onBrowseFiles?.(backup)}
              title="Browse Files"
            >
              <HiFolder className="w-4 h-4" />
            </Button>
          )}

          {/* Delete (only for completed/failed backups) */}
          {(backup.status === 'completed' || backup.status === 'failed') && onDelete && (
            <Button
              size="xs"
              color="failure"
              onClick={() => onDelete(backup)}
              title="Delete Backup"
            >
              <HiTrash className="w-4 h-4" />
            </Button>
          )}
        </div>
      </Table.Cell>
    </Table.Row>
  );
}
