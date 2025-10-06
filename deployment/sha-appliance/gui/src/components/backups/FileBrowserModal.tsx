'use client';

import { useState, useEffect } from 'react';
import { Modal, Button, Spinner, Alert, Table, Breadcrumb } from 'flowbite-react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { BackupJob, FileInfo } from '@/lib/types';
import {
  HiFolder,
  HiDocument,
  HiDownload,
  HiHome,
  HiChevronRight,
  HiExclamationCircle,
  HiInformationCircle
} from 'react-icons/hi';
import { formatDistanceToNow } from 'date-fns';

interface FileBrowserModalProps {
  isOpen: boolean;
  onClose: () => void;
  backup: BackupJob | null;
}

export function FileBrowserModal({ isOpen, onClose, backup }: FileBrowserModalProps) {
  const queryClient = useQueryClient();
  const [mountId, setMountId] = useState<string | null>(null);
  const [currentPath, setCurrentPath] = useState('/');
  const [error, setError] = useState<string | null>(null);

  // Mount backup mutation
  const mountBackupMutation = useMutation({
    mutationFn: (backupId: string) => api.mountBackup(backupId),
    onSuccess: (data) => {
      setMountId(data.mount_id);
      setError(null);
    },
    onError: (err: Error) => {
      setError(`Failed to mount backup: ${err.message}`);
    },
  });

  // Unmount backup mutation
  const unmountBackupMutation = useMutation({
    mutationFn: (mountId: string) => api.unmountBackup(mountId),
    onSuccess: () => {
      setMountId(null);
      queryClient.invalidateQueries({ queryKey: ['restore-mounts'] });
    },
  });

  // Fetch files for current path
  const { data: filesData, isLoading: filesLoading } = useQuery({
    queryKey: ['files', mountId, currentPath],
    queryFn: () => mountId ? api.listFiles(mountId, currentPath) : Promise.resolve({ files: [], total_count: 0 }),
    enabled: !!mountId && isOpen,
  });

  const files = filesData?.files || [];

  // Mount backup when modal opens
  useEffect(() => {
    if (isOpen && backup && !mountId && !mountBackupMutation.isPending) {
      mountBackupMutation.mutate(backup.backup_id);
    }
  }, [isOpen, backup, mountId]);

  // Unmount when modal closes
  const handleClose = () => {
    if (mountId) {
      unmountBackupMutation.mutate(mountId);
    }
    setMountId(null);
    setCurrentPath('/');
    setError(null);
    onClose();
  };

  // Navigate to directory
  const navigateToPath = (path: string) => {
    setCurrentPath(path);
  };

  // Navigate up one level
  const navigateUp = () => {
    if (currentPath === '/') return;
    const parts = currentPath.split('/').filter(Boolean);
    parts.pop();
    const newPath = parts.length > 0 ? '/' + parts.join('/') : '/';
    setCurrentPath(newPath);
  };

  // Format file size
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  // Get breadcrumb items
  const getBreadcrumbs = () => {
    if (currentPath === '/') return [{ name: 'Root', path: '/' }];
    
    const parts = currentPath.split('/').filter(Boolean);
    const breadcrumbs = [{ name: 'Root', path: '/' }];
    
    for (let i = 0; i < parts.length; i++) {
      const path = '/' + parts.slice(0, i + 1).join('/');
      breadcrumbs.push({ name: parts[i], path });
    }
    
    return breadcrumbs;
  };

  // Download file
  const handleDownloadFile = (file: FileInfo) => {
    if (!mountId) return;
    const url = api.getDownloadFileUrl(mountId, file.path);
    window.open(url, '_blank');
  };

  // Download directory
  const handleDownloadDirectory = (file: FileInfo) => {
    if (!mountId) return;
    const url = api.getDownloadDirectoryUrl(mountId, file.path, 'zip');
    window.open(url, '_blank');
  };

  return (
    <Modal show={isOpen} onClose={handleClose} size="4xl">
      <Modal.Header>
        <div className="flex flex-col">
          <span>Browse Backup Files</span>
          {backup && (
            <span className="text-sm font-normal text-gray-500 dark:text-gray-400 mt-1">
              {backup.vm_name} - {backup.backup_type} backup
            </span>
          )}
        </div>
      </Modal.Header>
      <Modal.Body>
        <div className="space-y-4">
          {/* Error Alert */}
          {error && (
            <Alert color="failure" icon={HiExclamationCircle}>
              {error}
            </Alert>
          )}

          {/* Mounting State */}
          {mountBackupMutation.isPending && (
            <div className="flex items-center justify-center p-8">
              <Spinner size="lg" />
              <span className="ml-3 text-gray-600 dark:text-gray-400">
                Mounting backup...
              </span>
            </div>
          )}

          {/* Mounted Successfully */}
          {mountId && (
            <>
              {/* Info Alert */}
              <Alert color="info" icon={HiInformationCircle}>
                <span className="text-sm">
                  Browse files and directories, then click download to restore individual files.
                  The backup will be automatically unmounted when you close this window.
                </span>
              </Alert>

              {/* Breadcrumb Navigation */}
              <div className="flex items-center justify-between border-b border-gray-200 dark:border-gray-700 pb-3">
                <Breadcrumb>
                  {getBreadcrumbs().map((crumb, index) => (
                    <Breadcrumb.Item
                      key={crumb.path}
                      onClick={() => index < getBreadcrumbs().length - 1 && navigateToPath(crumb.path)}
                      icon={index === 0 ? HiHome : undefined}
                      className={index < getBreadcrumbs().length - 1 ? 'cursor-pointer hover:text-blue-600' : ''}
                    >
                      {crumb.name}
                    </Breadcrumb.Item>
                  ))}
                </Breadcrumb>

                {currentPath !== '/' && (
                  <Button size="xs" color="gray" onClick={navigateUp}>
                    ↑ Up
                  </Button>
                )}
              </div>

              {/* File List */}
              {filesLoading ? (
                <div className="flex items-center justify-center p-4">
                  <Spinner />
                  <span className="ml-3 text-gray-600 dark:text-gray-400">
                    Loading files...
                  </span>
                </div>
              ) : files.length === 0 ? (
                <div className="text-center p-8">
                  <HiFolder className="w-12 h-12 text-gray-400 mx-auto mb-2" />
                  <p className="text-gray-600 dark:text-gray-400">
                    This directory is empty
                  </p>
                </div>
              ) : (
                <div className="overflow-x-auto max-h-96">
                  <Table>
                    <Table.Head>
                      <Table.HeadCell>Name</Table.HeadCell>
                      <Table.HeadCell>Size</Table.HeadCell>
                      <Table.HeadCell>Modified</Table.HeadCell>
                      <Table.HeadCell>Actions</Table.HeadCell>
                    </Table.Head>
                    <Table.Body className="divide-y">
                      {files.map((file) => (
                        <Table.Row
                          key={file.path}
                          className="bg-white dark:border-gray-700 dark:bg-gray-800"
                        >
                          {/* Name with Icon */}
                          <Table.Cell className="whitespace-nowrap font-medium text-gray-900 dark:text-white">
                            <div className="flex items-center">
                              {file.type === 'directory' ? (
                                <HiFolder className="w-5 h-5 text-blue-500 mr-2" />
                              ) : (
                                <HiDocument className="w-5 h-5 text-gray-500 mr-2" />
                              )}
                              {file.type === 'directory' ? (
                                <button
                                  onClick={() => navigateToPath(file.path)}
                                  className="text-blue-600 hover:underline dark:text-blue-400"
                                >
                                  {file.name}
                                </button>
                              ) : (
                                <span>{file.name}</span>
                              )}
                            </div>
                          </Table.Cell>

                          {/* Size */}
                          <Table.Cell>
                            {file.type === 'file' ? formatBytes(file.size) : '-'}
                          </Table.Cell>

                          {/* Modified */}
                          <Table.Cell>
                            <span className="text-sm text-gray-600 dark:text-gray-400">
                              {formatDistanceToNow(new Date(file.modified), { addSuffix: true })}
                            </span>
                          </Table.Cell>

                          {/* Actions */}
                          <Table.Cell>
                            {file.type === 'file' ? (
                              <Button
                                size="xs"
                                color="blue"
                                onClick={() => handleDownloadFile(file)}
                              >
                                <HiDownload className="w-4 h-4 mr-1" />
                                Download
                              </Button>
                            ) : (
                              <Button
                                size="xs"
                                color="purple"
                                onClick={() => handleDownloadDirectory(file)}
                              >
                                <HiDownload className="w-4 h-4 mr-1" />
                                Download ZIP
                              </Button>
                            )}
                          </Table.Cell>
                        </Table.Row>
                      ))}
                    </Table.Body>
                  </Table>
                </div>
              )}

              {/* File Count */}
              {files.length > 0 && (
                <div className="text-sm text-gray-600 dark:text-gray-400 text-center">
                  Showing {files.length} {files.length === 1 ? 'item' : 'items'}
                </div>
              )}
            </>
          )}
        </div>
      </Modal.Body>
      <Modal.Footer>
        <div className="flex justify-end w-full">
          <Button
            color="gray"
            onClick={handleClose}
            disabled={unmountBackupMutation.isPending}
          >
            {unmountBackupMutation.isPending ? 'Unmounting...' : 'Close'}
          </Button>
        </div>
      </Modal.Footer>
    </Modal>
  );
}

