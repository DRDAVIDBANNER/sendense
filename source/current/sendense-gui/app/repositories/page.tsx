"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Plus, RefreshCw } from "lucide-react";
import { PageHeader } from "@/components/common/PageHeader";
import { RepositoryCard, AddRepositoryModal, Repository, RepositoryCapacity } from "@/components/features/repositories";

// Helper function to transform backend repository data to GUI format
const transformRepository = (backendRepo: any): Repository => {
  // Convert bytes to GB
  const bytesToGB = (bytes: number) => Math.round(bytes / 1073741824);

  // Determine status based on enabled and usage percentage
  let status: 'online' | 'offline' | 'warning' = 'offline';
  if (backendRepo.enabled) {
    const usagePercent = backendRepo.storage?.used_percentage || 0;
    status = usagePercent > 85 ? 'warning' : 'online';
  }

  // Extract location from config based on type
  const getLocation = () => {
    const config = backendRepo.config || {};
    switch (backendRepo.type) {
      case 'local':
        return config.path || '';
      case 'nfs':
        return `${config.server || 'unknown'}:${config.export_path || ''}`;
      case 'cifs':
        return `\\\\${config.server || 'unknown'}\\${config.share_name || ''}`;
      case 's3':
        return `${config.bucket || 'unknown'} (${config.region || 'us-east-1'})`;
      case 'azure':
        return `${config.account_name || 'unknown'}/${config.container || ''}`;
      default:
        return 'Unknown';
    }
  };

  return {
    id: backendRepo.id,
    name: backendRepo.name,
    type: backendRepo.type,
    status,
    capacity: {
      total: bytesToGB(backendRepo.storage_info?.total_bytes || 0),
      used: bytesToGB(backendRepo.storage_info?.used_bytes || 0),
      available: bytesToGB(backendRepo.storage_info?.available_bytes || 0),
      unit: 'GB'
    },
    description: backendRepo.config?.description || undefined,
    lastTested: backendRepo.storage_info?.last_check_at || undefined,
    location: getLocation()
  };
};

export default function RepositoriesPage() {
  const [repositories, setRepositories] = useState<Repository[]>([]);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingRepository, setEditingRepository] = useState<Repository | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load repositories from backend API
  const loadRepositories = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/v1/repositories');
      const data = await response.json();

      if (!data.success) {
        throw new Error(data.error || 'Failed to load repositories');
      }

      // Transform backend data to GUI format
      const transformedRepos = data.repositories.map(transformRepository);
      setRepositories(transformedRepos);
    } catch (error) {
      console.error('Failed to load repositories:', error);
      setError('Failed to load repositories. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreateRepository = async (repositoryData: Omit<Repository, 'id' | 'status' | 'lastTested'>) => {
    try {
      // Build backend config object from repository data
      const buildConfig = () => {
        const config: any = {};

        // Extract location into proper config fields
        switch (repositoryData.type) {
          case 'local':
            config.path = repositoryData.location || '';
            break;
          case 'nfs':
            // Parse "server:/export/path" format
            const nfsParts = (repositoryData.location || '').split(':');
            config.server = nfsParts[0] || '';
            config.export_path = nfsParts[1] || '';
            break;
          case 'cifs':
            // Parse "\\server\share" format
            const cifsParts = (repositoryData.location || '').replace(/\\\\/g, '').split('\\');
            config.server = cifsParts[0] || '';
            config.share_name = cifsParts[1] || '';
            break;
          case 's3':
            // Parse "bucket (region)" format
            const s3Match = (repositoryData.location || '').match(/(.+?)\s*\((.+?)\)/);
            config.bucket = s3Match ? s3Match[1] : repositoryData.location;
            config.region = s3Match ? s3Match[2] : 'us-east-1';
            break;
          case 'azure':
            // Parse "account/container" format
            const azureParts = (repositoryData.location || '').split('/');
            config.account_name = azureParts[0] || '';
            config.container = azureParts[1] || '';
            break;
        }

        if (repositoryData.description) {
          config.description = repositoryData.description;
        }

        return config;
      };

      const requestBody = {
        name: repositoryData.name,
        type: repositoryData.type,
        enabled: true,
        is_immutable: false,
        config: buildConfig()
      };

      const response = await fetch('/api/v1/repositories', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody)
      });

      const data = await response.json();

      if (!data.success) {
        throw new Error(data.error || 'Failed to create repository');
      }

      // Close modal first for immediate feedback
      setIsAddModalOpen(false);
      
      // Reload repositories to get fresh data including the new one
      await loadRepositories();
      
      // Show success notification
      alert(`Repository "${repositoryData.name}" created successfully!`);
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
      const response = await fetch(`/api/v1/repositories/${repository.id}`, {
        method: 'DELETE'
      });

      const data = await response.json();

      if (!data.success) {
        // Backend returns specific error for repos with backups
        if (data.backup_count) {
          alert(`Cannot delete repository: ${data.backup_count} backups exist. Delete backups first.`);
        } else {
          throw new Error(data.error || 'Failed to delete repository');
        }
        return;
      }

      // Remove from local state
      setRepositories(prev => prev.filter(r => r.id !== repository.id));
    } catch (error) {
      console.error('Failed to delete repository:', error);
      alert('Failed to delete repository. See console for details.');
    }
  };

  const handleTestRepository = async (repository: Repository) => {
    try {
      // Backend test endpoint requires repository ID
      const response = await fetch(`/api/v1/repositories/${repository.id}/test`, {
        method: 'POST'
      });

      const data = await response.json();

      if (!data.success) {
        alert(`Connection test failed: ${data.error || 'Unknown error'}`);
        return;
      }

      // Update last tested timestamp on success
      setRepositories(prev => prev.map(r =>
        r.id === repository.id
          ? { ...r, lastTested: new Date().toISOString() }
          : r
      ));

      alert(`Connection test successful for "${repository.name}"`);
    } catch (error) {
      console.error('Failed to test repository:', error);
      alert('Connection test failed. See console for details.');
    }
  };

  const handleRefresh = async () => {
    setIsLoading(true);
    try {
      // Call backend refresh endpoint to update storage info for all repos
      const response = await fetch('/api/v1/repositories/refresh-storage', {
        method: 'POST'
      });

      const data = await response.json();

      if (!data.success) {
        throw new Error(data.error || 'Failed to refresh storage');
      }

      console.log(`Refreshed storage for ${data.refreshed_count} repositories`);
      if (data.failed_count > 0) {
        console.warn(`${data.failed_count} repositories failed to refresh`);
      }

      // Reload repositories to get updated storage info
      await loadRepositories();
    } catch (error) {
      console.error('Failed to refresh storage:', error);
      alert('Failed to refresh storage. See console for details.');
    } finally {
      setIsLoading(false);
    }
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

          {/* Error Display */}
          {error && (
            <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-4 mb-4">
              <p className="text-red-400">{error}</p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => { setError(null); loadRepositories(); }}
                className="mt-2"
              >
                Retry
              </Button>
            </div>
          )}

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

            {isLoading ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {[1, 2, 3].map(i => (
                  <Card key={i} className="animate-pulse">
                    <CardHeader>
                      <div className="h-6 bg-muted rounded w-3/4" />
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="h-4 bg-muted rounded w-1/2" />
                      <div className="h-2 bg-muted rounded w-full" />
                      <div className="h-4 bg-muted rounded w-2/3" />
                    </CardContent>
                  </Card>
                ))}
              </div>
            ) : repositories.length === 0 ? (
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
