"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Plus, RefreshCw } from "lucide-react";
import { PageHeader } from "@/components/common/PageHeader";
import { RepositoryCard, AddRepositoryModal, Repository, RepositoryCapacity } from "@/components/features/repositories";

// Mock repositories for demonstration
const mockRepositories: Repository[] = [
  {
    id: '1',
    name: 'Primary Local Storage',
    type: 'local',
    status: 'online',
    capacity: {
      total: 2000,
      used: 450,
      available: 1550,
      unit: 'GB'
    },
    description: 'Local SSD storage for critical backups',
    lastTested: '2025-10-06T10:00:00Z',
    location: '/mnt/primary-storage'
  },
  {
    id: '2',
    name: 'Cloud Backup - S3',
    type: 's3',
    status: 'online',
    capacity: {
      total: 50000,
      used: 12500,
      available: 37500,
      unit: 'GB'
    },
    description: 'Offsite cloud storage with encryption',
    lastTested: '2025-10-06T09:30:00Z',
    location: 'sendense-backups (us-east-1)'
  },
  {
    id: '3',
    name: 'NAS Archive',
    type: 'nfs',
    status: 'warning',
    capacity: {
      total: 10000,
      used: 8500,
      available: 1500,
      unit: 'GB'
    },
    description: 'Network attached storage for long-term retention',
    lastTested: '2025-10-05T14:00:00Z',
    location: 'nas-server:/export/archive'
  },
  {
    id: '4',
    name: 'File Server Share',
    type: 'cifs',
    status: 'offline',
    capacity: {
      total: 5000,
      used: 0,
      available: 0,
      unit: 'GB'
    },
    description: 'Windows file server backup destination',
    lastTested: '2025-10-04T08:00:00Z',
    location: '\\\\fileserver\\BackupShare'
  }
];

export default function RepositoriesPage() {
  const [repositories, setRepositories] = useState<Repository[]>(mockRepositories);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingRepository, setEditingRepository] = useState<Repository | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Simulate API calls
  const loadRepositories = async () => {
    setIsLoading(true);
    try {
      // In a real app, this would call the repository API
      // const response = await fetch('/api/v1/repositories');
      // const data = await response.json();
      // setRepositories(data);

      // For now, just simulate loading
      await new Promise(resolve => setTimeout(resolve, 1000));
      setRepositories(mockRepositories);
    } catch (error) {
      console.error('Failed to load repositories:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreateRepository = async (repositoryData: Omit<Repository, 'id' | 'status' | 'lastTested'>) => {
    try {
      // In a real app, this would call the create API
      // const response = await fetch('/api/v1/repositories', {
      //   method: 'POST',
      //   headers: { 'Content-Type': 'application/json' },
      //   body: JSON.stringify(repositoryData)
      // });
      // const newRepository = await response.json();

      // Mock implementation
      const newRepository: Repository = {
        ...repositoryData,
        id: Date.now().toString(),
        status: 'online',
        lastTested: new Date().toISOString()
      };

      setRepositories(prev => [...prev, newRepository]);
    } catch (error) {
      console.error('Failed to create repository:', error);
      throw error;
    }
  };

  const handleEditRepository = (repository: Repository) => {
    setEditingRepository(repository);
    setIsAddModalOpen(true);
  };

  const handleDeleteRepository = async (repository: Repository) => {
    if (!confirm(`Are you sure you want to delete repository "${repository.name}"?`)) {
      return;
    }

    try {
      // In a real app, this would call the delete API
      // await fetch(`/api/v1/repositories/${repository.id}`, { method: 'DELETE' });

      // Mock implementation
      setRepositories(prev => prev.filter(r => r.id !== repository.id));
    } catch (error) {
      console.error('Failed to delete repository:', error);
    }
  };

  const handleTestRepository = async (repository: Repository) => {
    try {
      // In a real app, this would call the test API
      // const response = await fetch(`/api/v1/repositories/${repository.id}/test`, {
      //   method: 'POST'
      // });
      // const result = await response.json();

      // Mock implementation - simulate testing
      console.log('Testing repository:', repository.name);

      // Update last tested timestamp
      setRepositories(prev => prev.map(r =>
        r.id === repository.id
          ? { ...r, lastTested: new Date().toISOString() }
          : r
      ));
    } catch (error) {
      console.error('Failed to test repository:', error);
    }
  };

  const handleRefresh = () => {
    loadRepositories();
  };

  useEffect(() => {
    loadRepositories();
  }, []);

  const getStatusCounts = () => {
    const counts = { online: 0, offline: 0, warning: 0 };
    repositories.forEach(repo => {
      counts[repo.status]++;
    });
    return counts;
  };

  const statusCounts = getStatusCounts();
  const totalCapacity = repositories.reduce((acc, repo) => acc + repo.capacity.total, 0);
  const totalUsed = repositories.reduce((acc, repo) => acc + repo.capacity.used, 0);

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Repository Management"
        breadcrumbs={[
          { label: "Dashboard", href: "/dashboard" },
          { label: "Repositories" }
        ]}
        actions={
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={handleRefresh}
              disabled={isLoading}
              className="gap-2"
            >
              <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
            <Button onClick={() => setIsAddModalOpen(true)} className="gap-2">
              <Plus className="h-4 w-4" />
              Add Repository
            </Button>
          </div>
        }
      />

      <div className="flex-1 overflow-hidden flex">
        {/* Main Content */}
        <div className="flex-1 flex flex-col min-w-0">
          {/* Summary Cards */}
          <div className="p-6 border-b border-border">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <div className="bg-card rounded-lg p-4 border border-border">
                <div className="text-2xl font-bold text-foreground">{repositories.length}</div>
                <div className="text-sm text-muted-foreground">Total Repositories</div>
              </div>
              <div className="bg-card rounded-lg p-4 border border-border">
                <div className="text-2xl font-bold text-green-600">{statusCounts.online}</div>
                <div className="text-sm text-muted-foreground">Online</div>
              </div>
              <div className="bg-card rounded-lg p-4 border border-border">
                <div className="text-2xl font-bold text-yellow-600">{statusCounts.warning}</div>
                <div className="text-sm text-muted-foreground">Warning</div>
              </div>
              <div className="bg-card rounded-lg p-4 border border-border">
                <div className="text-2xl font-bold text-red-600">{statusCounts.offline}</div>
                <div className="text-sm text-muted-foreground">Offline</div>
              </div>
            </div>

            {/* Capacity Summary */}
            <div className="mt-4 bg-card rounded-lg p-4 border border-border">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-lg font-semibold">Total Capacity</h3>
                <span className="text-sm text-muted-foreground">
                  {totalUsed} GB used of {totalCapacity} GB total
                </span>
              </div>
              <div className="w-full bg-secondary rounded-full h-3">
                <div
                  className="bg-primary h-3 rounded-full transition-all duration-300"
                  style={{ width: `${totalCapacity > 0 ? (totalUsed / totalCapacity) * 100 : 0}%` }}
                />
              </div>
            </div>
          </div>

          {/* Repository Grid */}
          <div className="flex-1 overflow-auto p-6">
            <div className="mb-6">
              <h2 className="text-lg font-semibold text-foreground mb-2">
                Storage Repositories
              </h2>
              <p className="text-muted-foreground">
                Manage and monitor your backup storage repositories across all locations
              </p>
            </div>

            {repositories.length === 0 ? (
              <div className="text-center py-12">
                <div className="text-muted-foreground mb-4">
                  No repositories configured yet
                </div>
                <Button onClick={() => setIsAddModalOpen(true)} className="gap-2">
                  <Plus className="h-4 w-4" />
                  Add Your First Repository
                </Button>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {repositories.map((repository) => (
                  <RepositoryCard
                    key={repository.id}
                    repository={repository}
                    onEdit={handleEditRepository}
                    onDelete={handleDeleteRepository}
                    onTest={handleTestRepository}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Add/Edit Repository Modal */}
      <AddRepositoryModal
        isOpen={isAddModalOpen}
        onClose={() => {
          setIsAddModalOpen(false);
          setEditingRepository(null);
        }}
        onCreate={handleCreateRepository}
        editingRepository={editingRepository}
      />
    </div>
  );
}
