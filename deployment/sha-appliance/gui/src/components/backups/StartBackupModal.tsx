'use client';

import { useState, useEffect } from 'react';
import { Modal, Button, Label, Select, TextInput, Alert } from 'flowbite-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { StartBackupRequest } from '@/lib/types';
import { HiInformationCircle, HiCheckCircle, HiExclamationCircle } from 'react-icons/hi';

interface StartBackupModalProps {
  isOpen: boolean;
  onClose: () => void;
  preselectedVmName?: string;
}

export function StartBackupModal({ isOpen, onClose, preselectedVmName }: StartBackupModalProps) {
  const queryClient = useQueryClient();
  
  // Form state
  const [vmName, setVmName] = useState(preselectedVmName || '');
  const [diskId, setDiskId] = useState('0');
  const [backupType, setBackupType] = useState<'full' | 'incremental'>('full');
  const [repositoryId, setRepositoryId] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  // Fetch VMs for dropdown
  const { data: vmContexts } = useQuery({
    queryKey: ['vm-contexts'],
    queryFn: () => api.getVMContexts(),
    enabled: isOpen,
  });

  // Fetch repositories (assuming there's an API endpoint)
  // For now, using a hardcoded list
  const repositories = [
    { id: 'local-repo-1', name: 'Local Repository' },
    { id: 'nfs-repo-1', name: 'NFS Repository' },
    { id: 'cifs-repo-1', name: 'CIFS Repository' },
  ];

  // Start backup mutation
  const startBackupMutation = useMutation({
    mutationFn: (request: StartBackupRequest) => api.startBackup(request),
    onSuccess: () => {
      setSuccess(true);
      setError(null);
      // Invalidate backups query to refresh the list
      queryClient.invalidateQueries({ queryKey: ['backups'] });
      queryClient.invalidateQueries({ queryKey: ['backups-all'] });
      
      // Close modal after 2 seconds
      setTimeout(() => {
        handleClose();
      }, 2000);
    },
    onError: (err: Error) => {
      setError(err.message);
      setSuccess(false);
    },
  });

  // Reset form when modal closes
  const handleClose = () => {
    setVmName(preselectedVmName || '');
    setDiskId('0');
    setBackupType('full');
    setRepositoryId('');
    setError(null);
    setSuccess(false);
    onClose();
  };

  // Handle form submission
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validation
    if (!vmName) {
      setError('Please select a VM');
      return;
    }
    if (!repositoryId) {
      setError('Please select a repository');
      return;
    }

    // Start backup
    startBackupMutation.mutate({
      vm_name: vmName,
      disk_id: parseInt(diskId),
      backup_type: backupType,
      repository_id: repositoryId,
    });
  };

  // Set default repository if none selected
  useEffect(() => {
    if (repositories.length > 0 && !repositoryId) {
      setRepositoryId(repositories[0].id);
    }
  }, [repositories, repositoryId]);

  return (
    <Modal show={isOpen} onClose={handleClose} size="md">
      <Modal.Header>Start Backup</Modal.Header>
      <Modal.Body>
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Success Message */}
          {success && (
            <Alert color="success" icon={HiCheckCircle}>
              <span className="font-medium">Backup started successfully!</span> The backup job is now running.
            </Alert>
          )}

          {/* Error Message */}
          {error && (
            <Alert color="failure" icon={HiExclamationCircle}>
              <span className="font-medium">Error:</span> {error}
            </Alert>
          )}

          {/* VM Selection */}
          <div>
            <Label htmlFor="vm-select" value="Virtual Machine" />
            <Select
              id="vm-select"
              value={vmName}
              onChange={(e) => setVmName(e.target.value)}
              required
              disabled={!!preselectedVmName}
            >
              <option value="">Select a VM...</option>
              {vmContexts?.contexts?.map((vm) => (
                <option key={vm.context_id} value={vm.vm_name}>
                  {vm.vm_name} ({vm.current_status})
                </option>
              ))}
            </Select>
          </div>

          {/* Disk ID */}
          <div>
            <Label htmlFor="disk-id" value="Disk Number" />
            <TextInput
              id="disk-id"
              type="number"
              min="0"
              max="15"
              value={diskId}
              onChange={(e) => setDiskId(e.target.value)}
              placeholder="0"
              required
              helperText="Disk number (0 for primary disk, 1+ for additional disks)"
            />
          </div>

          {/* Backup Type */}
          <div>
            <Label htmlFor="backup-type" value="Backup Type" />
            <Select
              id="backup-type"
              value={backupType}
              onChange={(e) => setBackupType(e.target.value as 'full' | 'incremental')}
              required
            >
              <option value="full">Full Backup</option>
              <option value="incremental">Incremental Backup</option>
            </Select>
            <div className="mt-2 text-xs text-gray-600 dark:text-gray-400">
              {backupType === 'full' ? (
                <div className="flex items-start">
                  <HiInformationCircle className="w-4 h-4 mr-1 mt-0.5 flex-shrink-0" />
                  <span>Full backup: Complete copy of the VM disk</span>
                </div>
              ) : (
                <div className="flex items-start">
                  <HiInformationCircle className="w-4 h-4 mr-1 mt-0.5 flex-shrink-0" />
                  <span>Incremental backup: Only changed blocks since last backup (requires CBT)</span>
                </div>
              )}
            </div>
          </div>

          {/* Repository Selection */}
          <div>
            <Label htmlFor="repository-select" value="Target Repository" />
            <Select
              id="repository-select"
              value={repositoryId}
              onChange={(e) => setRepositoryId(e.target.value)}
              required
            >
              <option value="">Select a repository...</option>
              {repositories.map((repo) => (
                <option key={repo.id} value={repo.id}>
                  {repo.name}
                </option>
              ))}
            </Select>
          </div>

          {/* Info Alert */}
          <Alert color="info" icon={HiInformationCircle}>
            <span className="text-sm">
              The backup will run in the background. You can monitor progress in the backup jobs list.
            </span>
          </Alert>
        </form>
      </Modal.Body>
      <Modal.Footer>
        <div className="flex justify-end space-x-2 w-full">
          <Button color="gray" onClick={handleClose} disabled={startBackupMutation.isPending}>
            Cancel
          </Button>
          <Button
            color="blue"
            onClick={handleSubmit}
            disabled={startBackupMutation.isPending || success}
            isProcessing={startBackupMutation.isPending}
          >
            {startBackupMutation.isPending ? 'Starting Backup...' : 'Start Backup'}
          </Button>
        </div>
      </Modal.Footer>
    </Modal>
  );
}

